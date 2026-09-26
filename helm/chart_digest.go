// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"helm.sh/helm/v3/pkg/action"
)

const chartDigestDescription = "Must be sha256: followed by 64 lowercase hexadecimal characters. SHA256 of a chart archive downloaded from an http or https URL."

const chartDigestInvalidMessage = "must be sha256: followed by exactly 64 lowercase hexadecimal characters"

var chartDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func chartDigestValidators() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(chartDigestPattern, chartDigestInvalidMessage),
	}
}

// validateChartDigest validates digest before the chart is downloaded.
func validateChartDigest(chartName, repositoryURL, digest string, digestUnknown bool) error {
	if digestUnknown {
		return fmt.Errorf("the digest attribute is unknown. A concrete sha256 digest is required before the chart can be downloaded")
	}
	if digest != "" {
		// buildChartNameWithRepository accepts any URI. Digest applies to http and https only.
		if !isHTTPURL(chartName) && !isHTTPURL(repositoryURL) {
			return fmt.Errorf("chart digest requires an http or https URL: digest checks the SHA256 of a chart archive downloaded from an http or https URL. OCI charts take @sha256 in the chart reference")
		}
		if err := validateDigestFormat(digest); err != nil {
			return err
		}
	}
	return nil
}

func isHTTPURL(ref string) bool {
	u, err := url.ParseRequestURI(ref)
	if err != nil || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func validateDigestFormat(raw string) error {
	if chartDigestPattern.MatchString(raw) {
		return nil
	}
	return fmt.Errorf("%s, got %q", chartDigestInvalidMessage, raw)
}

// downloadMaybePinnedChart locates a chart. When digest is set, a mismatch
// is refused.
func downloadMaybePinnedChart(meta *Meta, name string, cpo *action.ChartPathOptions, digest string) (string, error) {
	path, err := meta.LocateChart(cpo, name)
	if err != nil {
		return "", fmt.Errorf("unable to locate chart %s: %w", name, err)
	}
	if digest != "" {
		if err := validateDigestFormat(digest); err != nil {
			return "", fmt.Errorf("refusing to use chart %s: %w", name, err)
		}
		if err := verifyChartArchiveDigest(path, digest); err != nil {
			return "", fmt.Errorf("refusing to use chart %s: %w", name, err)
		}
	}
	return path, nil
}

func verifyChartArchiveDigest(path, digest string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat chart archive: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("digest verification requires a chart archive, but %s is a directory", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open chart archive: %w", err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("hash chart archive: %w", err)
	}
	got := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if got != digest {
		return fmt.Errorf("downloaded chart digest %s does not match pinned digest %s", got, digest)
	}
	return nil
}
