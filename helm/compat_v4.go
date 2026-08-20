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

// compatBySplitManifestsOrder sorts by in-file manifest order.
// In Helm v4, this moved to helm.sh/helm/v4/pkg/release/v1/util.
type compatBySplitManifestsOrder []string

func (a compatBySplitManifestsOrder) Len() int { return len(a) }
func (a compatBySplitManifestsOrder) Less(i, j int) bool {
	// Split `manifest-%d`
	anum, _ := strconvAtoi(a[i][len("manifest-"):])
	bnum, _ := strconvAtoi(a[j][len("manifest-"):])
	return anum < bnum
}
func (a compatBySplitManifestsOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// compatDependencies converts []*v2.Dependency to []chart.Dependency.
// In Helm v4, chart.Dependency is an alias for any.
func compatDependencies(deps []*v2.Dependency) []chart.Dependency {
	result := make([]chart.Dependency, 0, len(deps))
	for _, d := range deps {
		result = append(result, d)
	}
	return result
}

func strconvAtoi(s string) (int, error) {
	n := 0
	if len(s) == 0 {
		return 0, nil
	}
	sign := 1
	i := 0
	if s[0] == '-' {
		sign = -1
		i = 1
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, nil
		}
		n = n*10 + int(s[i]-'0')
	}
	return sign * n, nil
}
