package manifest

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
)

// CrdToManifests takes a Helm charts CRD and santizes the yaml before outputing
// the manifets in the same format as Helm template would.
func CrdToManifest(crds []chart.CRD) string {
	sb := strings.Builder{}
	for _, crd := range crds {
		split := releaseutil.SplitManifests(string(crd.File.Data))
		for _, v := range split {
			sb.WriteString(fmt.Sprintf("---\n# Source: %s\n%s\n", crd.Name, v))
		}
	}
	return sb.String()
}

// HooksToManifest converts Helm chart hooks to output in Helm template format.
func HooksToManifest(hooks []*release.Hook, skipTests bool) string {
	sb := strings.Builder{}
	for _, m := range hooks {
		if skipTests && isTestHook(m) {
			continue
		}
		sb.WriteString(fmt.Sprintf("---\n# Source: %s\n%s\n", m.Path, m.Manifest))
	}
	return sb.String()
}

// Compute takes the output from Helm template and converts it to a map.
func Compute(manifests string, showFiles []string) (map[string]string, string, error) {
	// Difference to the implementation of helm template in newTemplateCmd:
	// Independent of templates, names of the charts templates are always resolved from the manifests
	// to be able to populate the keys in the manifests computed attribute.
	var manifestsToRender []string

	splitManifests := releaseutil.SplitManifests(manifests)
	manifestsKeys := make([]string, 0, len(splitManifests))
	for k := range splitManifests {
		manifestsKeys = append(manifestsKeys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(manifestsKeys))

	// Mapping of manifest key to manifest template name
	manifestNamesByKey := make(map[string]string, len(manifestsKeys))

	manifestNameRegex := regexp.MustCompile("# Source: [^/]+/(.+)")

	for _, manifestKey := range manifestsKeys {
		manifest := splitManifests[manifestKey]
		submatch := manifestNameRegex.FindStringSubmatch(manifest)
		if len(submatch) == 0 {
			continue
		}
		manifestName := submatch[1]
		manifestNamesByKey[manifestKey] = manifestName
	}

	// if we have a list of files to render, then check that each of the
	// provided files exists in the chart.
	if len(showFiles) > 0 {
		for _, f := range showFiles {
			missing := true
			// Use linux-style filepath separators to unify user's input path
			f = filepath.ToSlash(f)
			for manifestKey, manifestName := range manifestNamesByKey {
				// manifest.Name is rendered using linux-style filepath separators on Windows as
				// well as macOS/linux.
				manifestPathSplit := strings.Split(manifestName, "/")
				// manifest.Path is connected using linux-style filepath separators on Windows as
				// well as macOS/linux
				manifestPath := strings.Join(manifestPathSplit, "/")

				// if the filepath provided matches a manifest path in the
				// chart, render that manifest
				if matched, _ := filepath.Match(f, manifestPath); !matched {
					continue
				}
				manifestsToRender = append(manifestsToRender, manifestKey)
				missing = false
			}

			if missing {
				return nil, "", fmt.Errorf("could not find template %q in chart", f)
			}
		}
	} else {
		manifestsToRender = manifestsKeys
	}

	// We need to sort the manifests so the order stays stable when they are
	// concatenated back together in the computedManifests map
	sort.Strings(manifestsToRender)

	// Map from rendered manifests to data source output
	computedManifests := make(map[string]string, 0)
	computedManifest := &strings.Builder{}

	for _, manifestKey := range manifestsToRender {
		manifest := splitManifests[manifestKey]
		manifestName := manifestNamesByKey[manifestKey]

		// Manifests
		computedManifests[manifestName] = fmt.Sprintf("%s---\n%s\n", computedManifests[manifestName], manifest)

		// Manifest bundle
		computedManifest.WriteString(fmt.Sprintf("---\n%s\n", manifest))
	}

	return computedManifests, computedManifest.String(), nil
}

func isTestHook(h *release.Hook) bool {
	for _, e := range h.Events {
		if e == release.HookTest {
			return true
		}
	}
	return false
}
