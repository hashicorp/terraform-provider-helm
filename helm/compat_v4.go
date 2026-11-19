// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/common"
	"helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/release/v1/util"
)

// compatChartutil provides shims for the chartutil package that was removed in Helm v4.

// ParseKubeVersion parses kubernetes version from string.
// In Helm v4, this moved to helm.sh/helm/v4/pkg/chart/common.
func compatParseKubeVersion(version string) (*common.KubeVersion, error) {
	return common.ParseKubeVersion(version)
}

// compatSplitManifests takes a manifest string and returns a map containing individual manifests.
// In Helm v4, this moved to helm.sh/helm/v4/pkg/release/v1/util.
func compatSplitManifests(bigFile string) map[string]string {
	return util.SplitManifests(bigFile)
}

// compatDependencies converts []*v2.Dependency to []chart.Dependency.
// In Helm v4, chart.Dependency is an alias for any.
func compatDependencies(deps []*v2.Dependency) []chart.Dependency {
	result := make([]chart.Dependency, 0, len(deps))
	for _, d := range deps {
		result = append(result, d)
	}
	return result
}
