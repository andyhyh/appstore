/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package search

import (
	"k8s.io/helm/cmd/helm/search"
	"sort"
)

func GetNewestVersion(packages []*search.Result) []*search.Result {
	newestVersions := make(map[string]*search.Result)
	for _, p := range packages {
		chartName := p.Chart.GetName()
		currChartVer := p.Chart.GetVersion()

		if newestVersions[chartName] == nil || currChartVer > newestVersions[chartName].Chart.GetVersion() {
			newestVersions[chartName] = p
		}

	}
	newestVersionsArray := make([]*search.Result, len(newestVersions))
	packageIdx := 0
	for _, v := range newestVersions {
		newestVersionsArray[packageIdx] = v
		packageIdx++
	}

	return newestVersionsArray
}

func GroupResultsByName(packages []*search.Result) [][]*search.Result {
	packageGroups := make(map[string][]*search.Result)
	var chartNames []string
	for _, res := range packages {
		chartName := res.Chart.GetName()
		packageGroups[chartName] = append(packageGroups[chartName], res)
	}

	for chartName, _ := range packageGroups {
		chartNames = append(chartNames, chartName)
	}

	allPackages := make([][]*search.Result, 0)
	sort.Strings(chartNames)

	for _, chartName := range chartNames {
		vals := packageGroups[chartName]
		SortByRevision(vals)
		allPackages = append(allPackages, vals)
	}

	return allPackages
}
