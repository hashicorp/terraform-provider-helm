// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// discoveryServer answers the legacy discovery endpoints of a cluster with core/v1
// ConfigMaps and apps/v1 Deployments, and counts the requests to /api and /apis.
// While hideApps is set, the cluster serves no apps group, as before a CRD install.
func discoveryServer(t *testing.T, hideApps *atomic.Bool) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var requests atomic.Int64
	responses := map[string]string{
		"/api":          `{"kind":"APIVersions","versions":["v1"]}`,
		"/apis":         `{"kind":"APIGroupList","apiVersion":"v1","groups":[{"name":"apps","versions":[{"groupVersion":"apps/v1","version":"v1"}],"preferredVersion":{"groupVersion":"apps/v1","version":"v1"}}]}`,
		"/api/v1":       `{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"configmaps","singularName":"configmap","namespaced":true,"kind":"ConfigMap","verbs":["get","list","create","delete"]}]}`,
		"/apis/apps/v1": `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"apps/v1","resources":[{"name":"deployments","singularName":"deployment","namespaced":true,"kind":"Deployment","verbs":["get","list","create","delete"]}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || r.URL.Path == "/apis" {
			requests.Add(1)
		}
		body, ok := responses[r.URL.Path]
		if hideApps.Load() {
			switch r.URL.Path {
			case "/apis":
				body = `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`
			case "/apis/apps/v1":
				ok = false
			}
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func testKubeConfig(server string) *KubeConfig {
	config := clientcmdapi.NewConfig()
	config.Clusters["test"] = &clientcmdapi.Cluster{Server: server}
	config.AuthInfos["test"] = &clientcmdapi.AuthInfo{}
	config.Contexts["test"] = &clientcmdapi.Context{Cluster: "test", AuthInfo: "test"}
	config.CurrentContext = "test"
	return &KubeConfig{ClientConfig: clientcmd.NewDefaultClientConfig(*config, nil), Burst: 100, QPS: 100}
}

// Helm asks the factory for a REST mapper for every object it builds. Each of
// these lookups must reuse the discovery of the first one: a release of 50
// objects otherwise fetches the API groups of the cluster 150 times.
func TestKubeConfigReusesDiscoveryAcrossRESTMappers(t *testing.T) {
	server, requests := discoveryServer(t, &atomic.Bool{})
	kubeConfig := testKubeConfig(server.URL)

	kinds := []schema.GroupKind{{Kind: "ConfigMap"}, {Group: "apps", Kind: "Deployment"}, {Kind: "ConfigMap"}}
	for _, kind := range kinds {
		mapper, err := kubeConfig.ToRESTMapper()
		if err != nil {
			t.Fatalf("ToRESTMapper: %v", err)
		}
		if _, err := mapper.RESTMapping(kind); err != nil {
			t.Fatalf("RESTMapping(%v): %v", kind, err)
		}
	}

	if got := requests.Load(); got != 2 {
		t.Errorf("discovery requests to /api and /apis = %d, want 2 (one discovery for all mappers)", got)
	}
}

// Helm invalidates the discovery client after it installs CRDs. A mapper it
// asks for afterwards must find the new kinds, not answer from the old cache.
func TestKubeConfigFindsKindsAddedAfterAnInvalidation(t *testing.T) {
	var hideApps atomic.Bool
	hideApps.Store(true)
	server, _ := discoveryServer(t, &hideApps)
	kubeConfig := testKubeConfig(server.URL)

	mapper, err := kubeConfig.ToRESTMapper()
	if err != nil {
		t.Fatalf("ToRESTMapper: %v", err)
	}
	if _, err := mapper.RESTMapping(schema.GroupKind{Kind: "ConfigMap"}); err != nil {
		t.Fatalf("RESTMapping: %v", err)
	}

	hideApps.Store(false)
	discoveryClient, err := kubeConfig.ToDiscoveryClient()
	if err != nil {
		t.Fatalf("ToDiscoveryClient: %v", err)
	}
	discoveryClient.Invalidate()
	mapper, err = kubeConfig.ToRESTMapper()
	if err != nil {
		t.Fatalf("ToRESTMapper: %v", err)
	}
	if _, err := mapper.RESTMapping(schema.GroupKind{Group: "apps", Kind: "Deployment"}); err != nil {
		t.Errorf("RESTMapping of a kind added after the invalidation: %v", err)
	}
}
