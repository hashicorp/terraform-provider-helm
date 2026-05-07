// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	openapi_v3 "github.com/google/gnostic-models/openapiv3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	gproto "google.golang.org/protobuf/proto"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi"
	kubeproto "k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/cmd/diff"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

// getKubeClient returns the underlying *kube.Client from an action.Configuration.
func getKubeClient(actionConfig *action.Configuration) (*kube.Client, error) {
	kc, ok := actionConfig.KubeClient.(*kube.Client)
	if !ok {
		return nil, errors.Errorf("client is not a *kube.Client")
	}
	return kc, nil
}

// gvkParser provides per-GroupVersion field-set parsers built lazily from
// the cluster's OpenAPI v3 schema.
type gvkParser struct {
	mu      sync.Mutex
	paths   map[string]openapi.GroupVersion
	parsers map[schema.GroupVersion]*managedfields.GvkParser
}

// newGVKParser builds a lazy per-GroupVersion parser cache from the cluster's
// OpenAPI v3 discovery client.
func newGVKParser(dc discovery.DiscoveryInterface) (*gvkParser, error) {
	paths, err := dc.OpenAPIV3().Paths()
	if err != nil {
		return nil, err
	}
	return &gvkParser{
		paths:   paths,
		parsers: map[schema.GroupVersion]*managedfields.GvkParser{},
	}, nil
}

func gvToAPIPath(gv schema.GroupVersion) string {
	if gv.Group == "" {
		return fmt.Sprintf("api/%s", gv.Version)
	}
	return fmt.Sprintf("apis/%s/%s", gv.Group, gv.Version)
}

// Type returns the parseable type for the given GVK, building the underlying
// per-GroupVersion parser on first use.
func (p *gvkParser) Type(gvk schema.GroupVersionKind) (*typed.ParseableType, error) {
	gv := gvk.GroupVersion()

	p.mu.Lock()
	defer p.mu.Unlock()

	parser, ok := p.parsers[gv]
	if !ok {
		path := gvToAPIPath(gv)
		gvc, ok := p.paths[path]
		if !ok {
			return nil, fmt.Errorf("no OpenAPI v3 schema for GroupVersion %q", gv.String())
		}
		bs, err := gvc.Schema(openapi.ContentTypeOpenAPIV3PB)
		if err != nil {
			return nil, fmt.Errorf("fetch OpenAPI v3 schema for %q: %w", gv.String(), err)
		}
		var doc openapi_v3.Document
		if err := gproto.Unmarshal(bs, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal OpenAPI v3 schema for %q: %w", gv.String(), err)
		}
		models, err := kubeproto.NewOpenAPIV3Data(&doc)
		if err != nil {
			return nil, err
		}
		normalizeV3Extensions(models)
		parser, err = managedfields.NewGVKParser(models, false)
		if err != nil {
			return nil, fmt.Errorf("build GVK parser for %q: %w", gv.String(), err)
		}
		p.parsers[gv] = parser
	}

	pt := parser.Type(gvk)
	if pt == nil {
		return nil, fmt.Errorf("no parseable type found for %s", gvk.String())
	}
	return pt, nil
}

// normalizeV3Extensions rewrites every model's vendor extensions so that
// managedfields.NewGVKParser (and the schemaconv it uses) can read them.
func normalizeV3Extensions(models kubeproto.Models) {
	for _, name := range models.ListModels() {
		m := models.LookupModel(name)
		if m == nil {
			continue
		}
		ext := m.GetExtensions()
		for k, v := range ext {
			ext[k] = toInterfaceKeyedMap(v)
		}
	}
}

// toInterfaceKeyedMap recursively converts map[string]interface{} values
// (yaml.v3 shape) to map[interface{}]interface{} (yaml.v2 shape), descending
// into nested maps and slices.
func toInterfaceKeyedMap(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[interface{}]interface{}, len(t))
		for k, vv := range t {
			out[k] = toInterfaceKeyedMap(vv)
		}
		return out
	case []interface{}:
		for i, vv := range t {
			t[i] = toInterfaceKeyedMap(vv)
		}
		return t
	default:
		return v
	}
}

// removeUnmanagedFields strips fields managed by kube-controller-manager or subresources.
func removeUnmanagedFields(parser *gvkParser, obj runtime.Object, gvk schema.GroupVersionKind) error {
	parseableType, err := parser.Type(gvk)
	if err != nil {
		return err
	}
	typedObj, err := parseableType.FromStructured(obj)
	if err != nil {
		return err
	}
	accessor, err := apimeta.Accessor(obj)
	if err != nil {
		return err
	}
	objManagedFields := accessor.GetManagedFields()
	fieldSet := &fieldpath.Set{}
	for _, mf := range objManagedFields {
		if mf.Manager == "kube-controller-manager" || mf.Subresource != "" {
			fs := &fieldpath.Set{}
			if err := fs.FromJSON(bytes.NewReader(mf.FieldsV1.Raw)); err != nil {
				return err
			}
			fieldSet = fieldSet.Union(fs)
		}
	}
	u := typedObj.RemoveItems(fieldSet).AsValue().Unstructured()
	m, ok := u.(map[string]interface{})
	if !ok {
		return errors.Errorf("unexpected type %T", u)
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(m, obj)
}

// mapRuntimeObjects converts runtime.Objects to JSON with unmanaged fields removed and sensitive values redacted.
func mapRuntimeObjects(ctx context.Context, kc *kube.Client, objects []runtime.Object) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	clientSet, err := kc.Factory.KubernetesClientSet()
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return nil, diags
	}
	parser, err := newGVKParser(clientSet.Discovery())
	if err != nil {
		diags.AddError("Parser Error", err.Error())
		return nil, diags
	}

	mappedObjects := make(map[string]string)
	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()

		if gvk.Kind == "Secret" {
			secret := &corev1.Secret{}
			if err := scheme.Scheme.Convert(obj, secret, nil); err != nil {
				diags.AddError("Secret Conversion Error", err.Error())
				return nil, diags
			}
			redactSecretData(secret)
			obj = secret
		}

		accessor, err := apimeta.Accessor(obj)
		if err != nil {
			diags.AddError("Object Access Error", err.Error())
			return nil, diags
		}

		key := fmt.Sprintf("%s/%s/%s/%s",
			strings.ToLower(gvk.GroupKind().String()),
			gvk.Version,
			accessor.GetNamespace(),
			accessor.GetName(),
		)

		if err := removeUnmanagedFields(parser, obj, gvk); err != nil {
			diags.AddError("Field Removal Error", err.Error())
			return nil, diags
		}

		accessor.SetUID(types.UID(""))
		accessor.SetCreationTimestamp(metav1.Time{})
		accessor.SetResourceVersion("")
		accessor.SetManagedFields(nil)

		if ta, err := apimeta.TypeAccessor(obj); err == nil {
			if ta.GetKind() == "" {
				ta.SetKind(gvk.Kind)
			}
			if ta.GetAPIVersion() == "" {
				ta.SetAPIVersion(gvk.GroupVersion().String())
			}
		}

		umap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			diags.AddError("Unstructured Conversion Error", err.Error())
			return nil, diags
		}
		normalizeK8sObject(umap)

		// Marshal back to JSON for the state
		objJSON, err := json.Marshal(umap)
		if err != nil {
			diags.AddError("Marshal Error", err.Error())
			return nil, diags
		}

		mappedObjects[key] = string(objJSON)
		tflog.Debug(ctx, "Mapped runtime object", map[string]interface{}{"key": key})
	}

	return mappedObjects, diags
}

func mapResources(ctx context.Context, actionConfig *action.Configuration, r *release.Release, f func(*resource.Info) (runtime.Object, error)) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	resources, err := actionConfig.KubeClient.Build(bytes.NewBufferString(r.Manifest), false)
	if err != nil {
		diags.AddError("Build Error", err.Error())
		return nil, diags
	}

	var objects []runtime.Object
	err = resources.Visit(func(i *resource.Info, err error) error {
		if err != nil {
			return err
		}
		obj, err := f(i)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		objects = append(objects, obj)
		return nil
	})
	if err != nil {
		diags.AddError("Visit Error", err.Error())
		return nil, diags
	}

	kc, err := getKubeClient(actionConfig)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return nil, diags
	}
	return mapRuntimeObjects(ctx, kc, objects)
}

// getLiveResources fetches the live cluster resources of a Helm release.
func getLiveResources(ctx context.Context, r *release.Release, m *Meta) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	actionConfig, err := m.GetHelmConfiguration(ctx, r.Namespace)
	if err != nil {
		diags.AddError("Helm Config Error", err.Error())
		return nil, diags
	}
	kc, err := getKubeClient(actionConfig)
	if err != nil {
		diags.AddError("Kube Client Error", err.Error())
		return nil, diags
	}
	rawResources, resDiags := mapResources(ctx, actionConfig, r, func(i *resource.Info) (runtime.Object, error) {
		gvk := i.Object.GetObjectKind().GroupVersionKind()
		return kc.Factory.NewBuilder().
			Unstructured().
			NamespaceParam(i.Namespace).DefaultNamespace().
			ResourceNames(gvk.GroupKind().String(), i.Name).
			Flatten().
			Do().
			Object()
	})
	diags.Append(resDiags...)
	if resDiags.HasError() {
		return rawResources, diags
	}

	cleaned := make(map[string]string, len(rawResources))
	for k, v := range rawResources {
		var obj map[string]any
		if err := json.Unmarshal([]byte(v), &obj); err != nil {
			cleaned[k] = v
			continue
		}
		normalizeK8sObject(obj)
		if b, err := json.Marshal(obj); err == nil {
			cleaned[k] = string(b)
		} else {
			cleaned[k] = v
		}
	}

	return cleaned, diags
}

func getDryRunResources(ctx context.Context, r *release.Release, m *Meta) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	actionConfig, err := m.GetHelmConfiguration(ctx, r.Namespace)
	if err != nil {
		diags.AddError("Helm Config Error", err.Error())
		return nil, diags
	}
	ioStreams := genericiooptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	fieldManager := "terraform-provider-helm"
	if os.Args[0] != "" {
		fieldManager = filepath.Base(os.Args[0])
	}

	rawResources, resDiags := mapResources(ctx, actionConfig, r, func(i *resource.Info) (runtime.Object, error) {
		info := &diff.InfoObject{
			LocalObj:        i.Object,
			Info:            i,
			Encoder:         scheme.DefaultJSONEncoder(),
			Force:           false,
			ServerSideApply: true,
			FieldManager:    fieldManager,
			ForceConflicts:  true,
			IOStreams:       ioStreams,
		}
		return info.Merged()
	})
	diags.Append(resDiags...)
	if resDiags.HasError() {
		return rawResources, diags
	}
	cleaned := make(map[string]string, len(rawResources))
	for k, v := range rawResources {
		var obj map[string]any
		if err := json.Unmarshal([]byte(v), &obj); err != nil {
			cleaned[k] = v
			continue
		}
		normalizeK8sObject(obj)
		if b, err := json.Marshal(obj); err == nil {
			cleaned[k] = string(b)
		} else {
			cleaned[k] = v
		}
	}

	return cleaned, diags
}
