// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/registry"
)

// newBasicAuthRegistry returns a minimal OCI registry that only answers the version check and the
// tag listing, and only to the given credentials.
func newBasicAuthRegistry(t *testing.T, username, password string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != username || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/v2/":
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/tags/list"):
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "charts/demo", "tags": []string{"1.0.0"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Logging into several OCI registries must keep the credentials of every one of them usable (#1719).
func TestOCIRegistryPerformLogin_multipleRegistries(t *testing.T) {
	t.Setenv("HELM_CONFIG_HOME", t.TempDir())

	regA := newBasicAuthRegistry(t, "user-a", "pass-a")
	regB := newBasicAuthRegistry(t, "user-b", "pass-b")

	client, err := registry.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	meta := &Meta{loggedInOCIRegistries: map[string]struct{}{}}
	meta.setRegistryClient(client)

	ctx := context.Background()
	for _, tc := range []struct {
		srv        *httptest.Server
		user, pass string
	}{
		{regA, "user-a", "pass-a"},
		{regB, "user-b", "pass-b"},
	} {
		if err := OCIRegistryPerformLogin(ctx, meta, tc.srv.URL, tc.user, tc.pass); err != nil {
			t.Fatalf("login to %s: %v", tc.srv.URL, err)
		}
	}

	for _, srv := range []*httptest.Server{regA, regB} {
		u, err := url.Parse(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		tags, err := meta.getRegistryClient().Tags(u.Host + "/charts/demo")
		if err != nil {
			t.Errorf("listing tags on %s after logging into both registries: %v", u.Host, err)
			continue
		}
		if len(tags) != 1 || tags[0] != "1.0.0" {
			t.Errorf("unexpected tags from %s: %v", u.Host, tags)
		}
	}
}
