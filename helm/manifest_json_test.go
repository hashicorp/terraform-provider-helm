// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertYAMLManifestToJSON(t *testing.T) {
	yamlManifest := readTestFile(t, "testdata/manifest_json/rendered_manifest.yaml")
	expectedJSON := readTestFile(t, "testdata/manifest_json/rendered_manifest.json")

	json, err := convertYAMLManifestToJSON(yamlManifest)

	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, json)
}

func readTestFile(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err.Error())
	}
	return string(b)
}
