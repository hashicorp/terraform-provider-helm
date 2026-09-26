// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"helm.sh/helm/v3/pkg/action"
)

func TestValidateChartDigestAllowsEmpty(t *testing.T) {
	if err := validateChartDigest("oci://registry.example.com/charts/app", "", "", false); err != nil {
		t.Fatal(err)
	}
}

func TestValidateChartDigestRejectsUnknown(t *testing.T) {
	if err := validateChartDigest("redis", "https://charts.example.com", "", true); err == nil {
		t.Fatal("expected an unknown digest to be rejected")
	}
}

func TestValidateChartDigestRejectsPaddedDigest(t *testing.T) {
	digest := " " + "sha256:" + strings.Repeat("ab", 32)
	if err := validateChartDigest("redis", "https://charts.example.com", digest, false); err == nil {
		t.Fatal("expected surrounding space in digest to be rejected")
	}
}

func TestValidateChartDigestRejectsBareHash(t *testing.T) {
	if err := validateChartDigest("redis", "https://charts.example.com", strings.Repeat("ab", 32), false); err == nil {
		t.Fatal("expected a bare hex hash to be rejected")
	}
}

func TestValidateChartDigestRejectsUpperHex(t *testing.T) {
	digest := "sha256:" + strings.Repeat("AB", 32)
	if err := validateChartDigest("redis", "https://charts.example.com", digest, false); err == nil {
		t.Fatal("expected uppercase hex in digest to be rejected")
	}
}

func TestChartPathOptionsRejectsOCIDigest(t *testing.T) {
	model := &HelmReleaseModel{
		Chart:      types.StringValue("postgresql"),
		Repository: types.StringValue("oci://registry-1.docker.io/bitnamicharts"),
		Version:    types.StringValue("16.5.0"),
		Digest:     types.StringValue("sha256:" + strings.Repeat("11", 32)),
	}
	_, _, diags := chartPathOptions(model, nil, &action.ChartPathOptions{})
	if !diags.HasError() {
		t.Fatal("expected digest on an OCI chart to be rejected")
	}
}

func TestChartPathOptionsAllowsHTTPDigest(t *testing.T) {
	digest := "sha256:" + strings.Repeat("44", 32)
	model := &HelmReleaseModel{
		Chart:      types.StringValue("redis"),
		Repository: types.StringValue("https://charts.example.com"),
		Version:    types.StringValue("6.0.1"),
		Digest:     types.StringValue(digest),
	}
	cpo, chartName, diags := chartPathOptions(model, nil, &action.ChartPathOptions{})
	if diags.HasError() {
		t.Fatalf("chartPathOptions: %v", diags)
	}
	if chartName != "redis" {
		t.Fatalf("chart = %q", chartName)
	}
	if cpo.RepoURL != "https://charts.example.com" {
		t.Fatalf("repository = %q", cpo.RepoURL)
	}
	if cpo.Version != "6.0.1" {
		t.Fatalf("version = %q", cpo.Version)
	}
}

func TestChartPathOptionsAllowsChartArchiveURL(t *testing.T) {
	digest := "sha256:" + strings.Repeat("55", 32)
	chartURL := "https://charts.example.com/redis-6.0.1.tgz"
	model := &HelmReleaseModel{
		Chart:   types.StringValue(chartURL),
		Version: types.StringValue("6.0.1"),
		Digest:  types.StringValue(digest),
	}
	cpo, chartName, diags := chartPathOptions(model, nil, &action.ChartPathOptions{})
	if diags.HasError() {
		t.Fatalf("chartPathOptions: %v", diags)
	}
	if chartName != chartURL {
		t.Fatalf("chart = %q", chartName)
	}
	if cpo.RepoURL != "" {
		t.Fatalf("repository = %q", cpo.RepoURL)
	}
	if cpo.Version != "" {
		t.Fatalf("version = %q", cpo.Version)
	}
}

func TestChartPathOptionsRejectsRepositoryName(t *testing.T) {
	model := &HelmReleaseModel{
		Chart:      types.StringValue("redis"),
		Repository: types.StringValue("bitnami"),
		Digest:     types.StringValue("sha256:" + strings.Repeat("66", 32)),
	}
	_, _, diags := chartPathOptions(model, nil, &action.ChartPathOptions{})
	if !diags.HasError() {
		t.Fatal("expected a repository name, rather than an http URL, to reject digest")
	}
}

func TestChartPathOptionsRejectsLocalDigest(t *testing.T) {
	model := &HelmReleaseModel{
		Chart:   types.StringValue("./testdata/charts/test-chart"),
		Version: types.StringValue("0.1.0"),
		Digest:  types.StringValue("sha256:" + strings.Repeat("22", 32)),
	}
	_, _, diags := chartPathOptions(model, nil, &action.ChartPathOptions{})
	if !diags.HasError() {
		t.Fatal("expected local charts to reject digest")
	}
}

func TestChartPathOptionsTemplateRejectsOCIDigest(t *testing.T) {
	model := &HelmTemplateModel{
		Chart:      types.StringValue("postgresql"),
		Repository: types.StringValue("oci://registry-1.docker.io/bitnamicharts"),
		Version:    types.StringValue("16.5.0"),
		Digest:     types.StringValue("sha256:" + strings.Repeat("33", 32)),
	}
	_, _, diags := chartPathOptionsModel(model, nil, &action.ChartPathOptions{})
	if !diags.HasError() {
		t.Fatal("expected digest on an OCI chart to be rejected")
	}
}

func TestVerifyChartArchiveDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chart.tgz")
	body := []byte("chart-bytes")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(body)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if err := verifyChartArchiveDigest(path, "sha256:"+strings.Repeat("00", 32)); err == nil {
		t.Fatal("expected digest mismatch")
	}
	if err := verifyChartArchiveDigest(path, digest); err != nil {
		t.Fatalf("matching digest rejected: %s", err)
	}
}

func TestUpgradeReleaseStateAddsNullDigest(t *testing.T) {
	upgraded, err := upgradeHelmReleaseStateAddDigest(&tfprotov6.RawState{
		JSON: []byte(`{"name":"demo","version":"1.2.3"}`),
	})
	if err != nil {
		t.Fatalf("upgrade: %s", err)
	}
	var got map[string]any
	if err := json.Unmarshal(upgraded.JSON, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "demo" || got["version"] != "1.2.3" {
		t.Fatalf("state = %#v", got)
	}
	digest, ok := got["digest"]
	if !ok || digest != nil {
		t.Fatalf("digest = %#v, present=%v, want null", digest, ok)
	}
}

func TestUpgradeReleaseStateKeepsDigest(t *testing.T) {
	want := "sha256:" + strings.Repeat("ab", 32)
	upgraded, err := upgradeHelmReleaseStateAddDigest(&tfprotov6.RawState{
		JSON: []byte(`{"name":"demo","digest":"` + want + `"}`),
	})
	if err != nil {
		t.Fatalf("upgrade: %s", err)
	}
	var got map[string]any
	if err := json.Unmarshal(upgraded.JSON, &got); err != nil {
		t.Fatal(err)
	}
	if got["digest"] != want {
		t.Fatalf("digest = %#v", got["digest"])
	}
}

func TestRecomputeMetadataOnDigestChange(t *testing.T) {
	setType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
		"type":  types.StringType,
	}}
	setListType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":  types.StringType,
		"value": types.ListType{ElemType: types.StringType},
	}}
	base := HelmReleaseModel{
		Chart:        types.StringValue("redis"),
		Repository:   types.StringValue("https://charts.example.com"),
		Version:      types.StringValue("6.0.1"),
		Digest:       types.StringValue("sha256:" + strings.Repeat("ab", 32)),
		Values:       types.ListNull(types.StringType),
		Set:          types.ListNull(setType),
		SetSensitive: types.ListNull(setType),
		SetList:      types.ListNull(setListType),
	}
	state := base
	if recomputeMetadata(base, &state) {
		t.Fatal("unchanged digest should not recompute metadata")
	}
	base.Digest = types.StringValue("sha256:" + strings.Repeat("cd", 32))
	if !recomputeMetadata(base, &state) {
		t.Fatal("digest change should recompute metadata")
	}
}
