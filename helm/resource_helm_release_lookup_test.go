// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"helm.sh/helm/v3/pkg/cli"
)

// unreachableMeta is a provider configuration whose cluster cannot be reached.
// Every release lookup against it fails with a connection error, never with
// "release: not found", which is the situation of an API server behind an IP
// allow list, an expired credential or a cluster that is slow to answer.
func unreachableMeta(t *testing.T) *Meta {
	t.Helper()

	attrTypes := map[string]attr.Type{
		"host":                     types.StringType,
		"username":                 types.StringType,
		"password":                 types.StringType,
		"insecure":                 types.BoolType,
		"tls_server_name":          types.StringType,
		"client_certificate":       types.StringType,
		"client_key":               types.StringType,
		"cluster_ca_certificate":   types.StringType,
		"config_paths":             types.ListType{ElemType: types.StringType},
		"config_path":              types.StringType,
		"config_context":           types.StringType,
		"config_context_auth_info": types.StringType,
		"config_context_cluster":   types.StringType,
		"token":                    types.StringType,
		"proxy_url":                types.StringType,
		"exec":                     types.ObjectType{AttrTypes: execSchemaAttrTypes()},
	}
	values := map[string]attr.Value{
		"host":                     types.StringValue("https://127.0.0.1:1"),
		"username":                 types.StringNull(),
		"password":                 types.StringNull(),
		"insecure":                 types.BoolValue(true),
		"tls_server_name":          types.StringNull(),
		"client_certificate":       types.StringNull(),
		"client_key":               types.StringNull(),
		"cluster_ca_certificate":   types.StringNull(),
		"config_paths":             types.ListNull(types.StringType),
		"config_path":              types.StringNull(),
		"config_context":           types.StringNull(),
		"config_context_auth_info": types.StringNull(),
		"config_context_cluster":   types.StringNull(),
		"token":                    types.StringNull(),
		"proxy_url":                types.StringNull(),
		"exec":                     types.ObjectNull(execSchemaAttrTypes()),
	}
	kubernetes, diags := types.ObjectValue(attrTypes, values)
	if diags.HasError() {
		t.Fatalf("building kubernetes config: %v", diags)
	}

	return &Meta{
		Data: &HelmProviderModel{
			Kubernetes: kubernetes,
			HelmDriver: types.StringValue("secret"),
		},
		Settings:   cli.New(),
		HelmDriver: "secret",
	}
}

// releaseState is the state of one helm_release, with every attribute null
// except its name and namespace.
func releaseState(t *testing.T, ctx context.Context, r *HelmRelease) tfsdk.State {
	t.Helper()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResp.Diagnostics)
	}

	objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["id"] = tftypes.NewValue(tftypes.String, "example")
	values["name"] = tftypes.NewValue(tftypes.String, "example")
	values["namespace"] = tftypes.NewValue(tftypes.String, "default")

	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objectType, values),
	}
}

// A lookup that fails is not a release that is gone: Read must return the
// error and keep the release in the state. Before the fix it removed the
// release, and the next plan re-created one that still existed, which Helm
// refuses with "cannot re-use a name that is still in use" (#1669).
func TestHelmReleaseRead_lookupFailureKeepsState(t *testing.T) {
	ctx := context.Background()
	r := &HelmRelease{meta: unreachableMeta(t)}
	state := releaseState(t, ctx, r)

	resp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the read of an unreachable cluster to fail, got no error")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "Error checking release existence") {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors()[0].Summary())
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("the release was removed from the state although the lookup failed")
	}
}

// The same for Delete: a failed lookup must fail the delete, not skip the
// uninstall and drop a release that still exists from the state.
func TestHelmReleaseDelete_lookupFailureFails(t *testing.T) {
	ctx := context.Background()
	r := &HelmRelease{meta: unreachableMeta(t)}
	state := releaseState(t, ctx, r)

	resp := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the delete against an unreachable cluster to fail, got no error")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "Error checking release existence") {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors()[0].Summary())
	}
}
