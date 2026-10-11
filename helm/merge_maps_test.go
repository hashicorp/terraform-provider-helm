package helm

import (
	"reflect"
	"testing"
)

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]interface{}
		b    map[string]interface{}
		want map[string]interface{}
	}{
		{
			name: "both empty",
			a:    map[string]interface{}{},
			b:    map[string]interface{}{},
			want: map[string]interface{}{},
		},
		{
			name: "a empty, b has keys",
			a:    map[string]interface{}{},
			b:    map[string]interface{}{"x": 1},
			want: map[string]interface{}{"x": 1},
		},
		{
			name: "a has keys, b empty",
			a:    map[string]interface{}{"x": 1},
			b:    map[string]interface{}{},
			want: map[string]interface{}{"x": 1},
		},
		{
			name: "non-overlapping keys",
			a:    map[string]interface{}{"a": 1},
			b:    map[string]interface{}{"b": 2},
			want: map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name: "overlapping scalar key, b wins",
			a:    map[string]interface{}{"k": "old"},
			b:    map[string]interface{}{"k": "new"},
			want: map[string]interface{}{"k": "new"},
		},
		{
			name: "both values are maps, recursive merge",
			a:    map[string]interface{}{"nested": map[string]interface{}{"x": 1, "y": 2}},
			b:    map[string]interface{}{"nested": map[string]interface{}{"y": 3, "z": 4}},
			want: map[string]interface{}{"nested": map[string]interface{}{"x": 1, "y": 3, "z": 4}},
		},
		{
			name: "a is map, b is scalar, b wins",
			a:    map[string]interface{}{"k": map[string]interface{}{"x": 1}},
			b:    map[string]interface{}{"k": "scalar"},
			want: map[string]interface{}{"k": "scalar"},
		},
		{
			name: "a is scalar, b is map, b wins",
			a:    map[string]interface{}{"k": "scalar"},
			b:    map[string]interface{}{"k": map[string]interface{}{"x": 1}},
			want: map[string]interface{}{"k": map[string]interface{}{"x": 1}},
		},
		{
			name: "deeply nested merge",
			a:    map[string]interface{}{"l1": map[string]interface{}{"l2": map[string]interface{}{"a": 1, "b": 2}}},
			b:    map[string]interface{}{"l1": map[string]interface{}{"l2": map[string]interface{}{"b": 3, "c": 4}}},
			want: map[string]interface{}{"l1": map[string]interface{}{"l2": map[string]interface{}{"a": 1, "b": 3, "c": 4}}},
		},
		{
			name: "a nil, b has keys",
			a:    nil,
			b:    map[string]interface{}{"x": 1},
			want: map[string]interface{}{"x": 1},
		},
		{
			name: "a has keys, b nil",
			a:    map[string]interface{}{"x": 1},
			b:    nil,
			want: map[string]interface{}{"x": 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeMaps(tc.a, tc.b)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("mergeMaps() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestMergeMaps_SliceBreak(t *testing.T) {
	a := map[string]interface{}{
		"list": []interface{}{
			map[string]interface{}{"x": 1, "y": 2},
		},
	}
	b := map[string]interface{}{
		"list": []interface{}{
			map[string]interface{}{"y": 3},
		},
	}

	want := map[string]interface{}{
		"list": []interface{}{
			map[string]interface{}{"x": 1, "y": 3},
		},
	}

	got := mergeMaps(a, b)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
