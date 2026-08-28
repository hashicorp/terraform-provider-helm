package helm

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A PEM bundle shaped like the ones distributed as a trust store: comment headers, one of which
// carries a comma, and base64 bodies whose padding contains "=". The "=" is what makes this case
// dangerous -- the text after the comma parses as a valid assignment, so strvals returns no error
// and the caller is handed a truncated value.
const bundleWithCommaInHeader = `GlobalSign Root CA, R3
======================
-----BEGIN CERTIFICATE-----
MIIBfake+base64/data==
-----END CERTIFICATE-----
`

func setEntry(name, value, valueType string) setResourceModel {
	return setResourceModel{
		Name:  types.StringValue(name),
		Value: types.StringValue(value),
		Type:  types.StringValue(valueType),
	}
}

func TestGetValueRejectsSplitValue(t *testing.T) {
	for _, tc := range []struct {
		name      string
		valueType string
	}{
		{"string", "string"},
		{"auto", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := map[string]interface{}{}
			diags := getValue(base, setEntry("secrets.ca", bundleWithCommaInHeader, tc.valueType), false)

			if !diags.HasError() {
				t.Fatalf("expected an error, got none; base parsed to %#v", base)
			}
			if len(base) != 0 {
				t.Errorf("expected nothing to be written on failure, got %#v", base)
			}

			summary := diags.Errors()[0].Summary()
			detail := diags.Errors()[0].Detail()
			if summary != "Value would be truncated" {
				t.Errorf("unexpected summary %q", summary)
			}
			// The value may be a secret, so it must not appear in the diagnostic.
			if strings.Contains(detail, "MIIBfake") || strings.Contains(summary, "MIIBfake") {
				t.Errorf("diagnostic leaked the value: %s / %s", summary, detail)
			}
		})
	}
}

// Without the guard this is the silent failure: strvals returns no error, the key keeps only the
// text before the comma, and the rest lands as unrelated keys.
func TestStrvalsSilentlyTruncatesUnescapedComma(t *testing.T) {
	base := map[string]interface{}{}
	if d := checkValueIsNotSplit("secrets.ca", bundleWithCommaInHeader, true); d == nil {
		t.Fatal("guard did not fire on a value that strvals splits")
	}

	// Demonstrate the behaviour the guard exists to prevent.
	diags := getValue(base, setEntry("secrets.ca", strings.ReplaceAll(bundleWithCommaInHeader, ",", `\,`), "string"), false)
	if diags.HasError() {
		t.Fatalf("escaped value should parse: %v", diags.Errors())
	}
	secrets, ok := base["secrets"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected secrets map, got %#v", base)
	}
	if got := secrets["ca"]; got != bundleWithCommaInHeader {
		t.Errorf("escaped value did not round-trip:\n got %q\nwant %q", got, bundleWithCommaInHeader)
	}
	if len(base) != 1 || len(secrets) != 1 {
		t.Errorf("expected exactly one key, got %#v", base)
	}
}

func TestGetValueAcceptsValuesThatAreNotSplit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		key       string
		value     string
		valueType string
		want      interface{}
	}{
		{"plain string", "a", "no commas here", "string", "no commas here"},
		{"escaped comma", "a", `x\,y`, "string", "x,y"},
		{"nested key", "a.b.c", "v", "string", nil},
		{"multiline without commas", "a", "line1\nline2", "string", "line1\nline2"},
		{"equals in value", "a", "key=value", "string", "key=value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := map[string]interface{}{}
			diags := getValue(base, setEntry(tc.key, tc.value, tc.valueType), false)
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags.Errors())
			}
			if tc.want != nil {
				if got := base[tc.key]; got != tc.want {
					t.Errorf("got %#v, want %#v", got, tc.want)
				}
			}
		})
	}
}

// Brace syntax is how a list is meant to be passed through set, and its commas are internal to the
// list rather than assignment separators, so it must keep working.
func TestGetValueAcceptsBraceListSyntax(t *testing.T) {
	base := map[string]interface{}{}
	diags := getValue(base, setEntry("a", "{x,y,z}", "string"), false)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags.Errors())
	}
	list, ok := base["a"].([]interface{})
	if !ok {
		t.Fatalf("expected a list, got %#v", base["a"])
	}
	if len(list) != 3 {
		t.Errorf("expected 3 elements, got %#v", list)
	}
}

func TestCountLeaves(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   map[string]interface{}
		want int
	}{
		{"empty", map[string]interface{}{}, 0},
		{"one leaf", map[string]interface{}{"a": "b"}, 1},
		{"nested leaf", map[string]interface{}{"a": map[string]interface{}{"b": "c"}}, 1},
		{"list counts once", map[string]interface{}{"a": []interface{}{"x", "y"}}, 1},
		{"two leaves", map[string]interface{}{"a": "b", "c": "d"}, 2},
		{"nested and sibling", map[string]interface{}{"a": map[string]interface{}{"b": "c"}, "d": "e"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := countLeaves(tc.in); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCheckListIsNotSplit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		key       string
		elements  []string
		wantError bool
	}{
		{"clean elements", "a", []string{"x", "y"}, false},
		{"element with comma", "a", []string{"x,y", "z"}, true},
		{"escaped comma", "a", []string{`x\,y`, "z"}, false},
		{"nested key", "a.b", []string{"x", "y"}, false},
		{"single element with comma", "a", []string{"x,y"}, true},
		{"empty list", "a", []string{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := checkListIsNotSplit(tc.key, tc.elements)
			if tc.wantError && d == nil {
				t.Fatal("expected an error, got none")
			}
			if !tc.wantError && d != nil {
				t.Fatalf("unexpected error: %s / %s", d.Summary(), d.Detail())
			}
			if d != nil && strings.Contains(d.Detail(), "x,y") {
				t.Errorf("diagnostic leaked an element: %s", d.Detail())
			}
		})
	}
}

func TestLookupPath(t *testing.T) {
	m := map[string]interface{}{"a": map[string]interface{}{"b": []interface{}{"x"}}, "c": "d"}
	if got := lookupPath(m, "a.b"); len(got.([]interface{})) != 1 {
		t.Errorf("a.b: got %#v", got)
	}
	if got := lookupPath(m, "c"); got != "d" {
		t.Errorf("c: got %#v", got)
	}
	for _, missing := range []string{"a.z", "z", "c.d"} {
		if got := lookupPath(m, missing); got != nil {
			t.Errorf("%s: expected nil, got %#v", missing, got)
		}
	}
}

// The parser quotes the fragment it choked on, which for a sensitive entry is part of the secret.
// Diagnostics are shown in plan output, so that fragment must not reach them.
func TestGetValueDoesNotLeakSensitiveValueInParseError(t *testing.T) {
	// The tail after the comma has no "=", so strvals errors rather than parsing it as another
	// assignment -- the path that surfaces the underlying parser error.
	const secret = "topsecret, MORE-SECRET-MATERIAL"

	t.Run("sensitive withholds the parser error", func(t *testing.T) {
		base := map[string]interface{}{}
		diags := getValue(base, setEntry("secrets.token", secret, "string"), true)
		if !diags.HasError() {
			t.Fatal("expected an error")
		}
		for _, d := range diags.Errors() {
			if strings.Contains(d.Detail(), "MORE-SECRET-MATERIAL") || strings.Contains(d.Summary(), "MORE-SECRET-MATERIAL") {
				t.Errorf("secret material reached the diagnostic: %s / %s", d.Summary(), d.Detail())
			}
			if !strings.Contains(d.Detail(), "secrets.token") {
				t.Errorf("diagnostic should still name the key: %s", d.Detail())
			}
		}
	})

	t.Run("non-sensitive keeps the parser error", func(t *testing.T) {
		base := map[string]interface{}{}
		diags := getValue(base, setEntry("a.b", secret, "string"), false)
		if !diags.HasError() {
			t.Fatal("expected an error")
		}
		if !strings.Contains(diags.Errors()[0].Detail(), "has no value") {
			t.Errorf("non-sensitive entries should keep the underlying error: %s", diags.Errors()[0].Detail())
		}
	})
}
