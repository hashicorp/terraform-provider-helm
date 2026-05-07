// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGVToAPIPath(t *testing.T) {
	tests := []struct {
		name string
		gv   schema.GroupVersion
		want string
	}{
		{
			name: "core group uses /api",
			gv:   schema.GroupVersion{Group: "", Version: "v1"},
			want: "api/v1",
		},
		{
			name: "named group uses /apis",
			gv:   schema.GroupVersion{Group: "apps", Version: "v1"},
			want: "apis/apps/v1",
		},
		{
			name: "dotted group",
			gv:   schema.GroupVersion{Group: "cert-manager.io", Version: "v1"},
			want: "apis/cert-manager.io/v1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, gvToAPIPath(tc.gv))
		})
	}
}

func TestToInterfaceKeyedMap(t *testing.T) {
	t.Run("scalars are returned unchanged", func(t *testing.T) {
		for _, v := range []interface{}{"x", 1, 1.5, true, nil} {
			assert.Equal(t, v, toInterfaceKeyedMap(v))
		}
	})

	t.Run("string-keyed map is converted to interface-keyed map", func(t *testing.T) {
		in := map[string]interface{}{"a": "b"}
		out := toInterfaceKeyedMap(in)
		want := map[interface{}]interface{}{"a": "b"}
		assert.Equal(t, want, out)
	})

	t.Run("nested maps are converted recursively", func(t *testing.T) {
		in := map[string]interface{}{
			"outer": map[string]interface{}{
				"inner": map[string]interface{}{"k": "v"},
			},
		}
		out := toInterfaceKeyedMap(in)
		want := map[interface{}]interface{}{
			"outer": map[interface{}]interface{}{
				"inner": map[interface{}]interface{}{"k": "v"},
			},
		}
		assert.Equal(t, want, out)
	})

	t.Run("maps inside slices are converted", func(t *testing.T) {
		in := []interface{}{
			map[string]interface{}{"group": "", "version": "v1", "kind": "Pod"},
			map[string]interface{}{"group": "apps", "version": "v1", "kind": "Deployment"},
		}
		out := toInterfaceKeyedMap(in)
		want := []interface{}{
			map[interface{}]interface{}{"group": "", "version": "v1", "kind": "Pod"},
			map[interface{}]interface{}{"group": "apps", "version": "v1", "kind": "Deployment"},
		}
		assert.Equal(t, want, out)
	})

	t.Run("already interface-keyed map is returned unchanged", func(t *testing.T) {
		in := map[interface{}]interface{}{"a": "b"}
		out := toInterfaceKeyedMap(in)
		assert.Equal(t, in, out)
	})

	t.Run("slice is mutated in place and returned", func(t *testing.T) {
		in := []interface{}{map[string]interface{}{"a": "b"}}
		out := toInterfaceKeyedMap(in)
		// same backing slice
		assert.True(t, reflect.ValueOf(in).Pointer() == reflect.ValueOf(out).Pointer())
		// element was converted
		assert.Equal(t, map[interface{}]interface{}{"a": "b"}, in[0])
	})
}
