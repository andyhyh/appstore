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
	"fmt"
	"github.com/Masterminds/semver"
	log "github.com/Sirupsen/logrus"

	"k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/repo"
)

// searchMaxScore suggests that any score higher than this is not considered a match.
const searchMaxScore = 25

var index *search.Index

func ensureIndex(settings *helm_env.EnvSettings) error {
	if index == nil {
		newIndex, err := buildIndex(settings)
		index = newIndex
		if err != nil {
			return err
		}
	}

	return nil
}

func GetAllCharts(settings *helm_env.EnvSettings) ([]*search.Result, error) {
	err := ensureIndex(settings)
	if err != nil {
		return nil, err
	}

	res := index.All()
	return res, nil
}

func SearchCharts(settings *helm_env.EnvSettings, query string, version string) ([]*search.Result, error) {
	err := ensureIndex(settings)
	if err != nil {
		return nil, err
	}

	var res []*search.Result
	if len(query) == 0 {
		res = index.All()
	} else {
		res, err = index.Search(query, searchMaxScore, false)
		if err != nil {
			return nil, err
		}
	}

	search.SortScore(res)
	data, err := applyConstraint(res, version)

	if err != nil {
		return nil, err
	}

	return data, err
}

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

func GroupPackages(packages []*search.Result) map[string][]*search.Result {
	packageGroups := make(map[string][]*search.Result)
	for _, res := range packages {
		chartName := res.Chart.GetName()
		packageGroups[chartName] = append(packageGroups[chartName], res)
	}

	for _, v := range packageGroups {
		SortByRevision(v)
	}

	return packageGroups
}

func GetSinglePackage(settings *helm_env.EnvSettings, packageName string) ([]*search.Result, error) {
	err := ensureIndex(settings)
	if err != nil {
		return nil, err
	}

	allPackages := index.All()

	results := []*search.Result{}
	for _, p := range allPackages {
		if p.Chart.GetName() == packageName {
			results = append(results, p)
		}
	}

	if len(results) == 0 {
		log.Warn("Package not found!")
		return nil, nil
	}
	SortByRevision(results)
	return results, err
}

func applyConstraint(res []*search.Result, version string) ([]*search.Result, error) {
	if len(version) == 0 {
		return res, nil
	}

	constraint, err := semver.NewConstraint(version)
	if err != nil {
		return res, fmt.Errorf("an invalid version/constraint format: %s", err)
	}

	data := res[:0]
	for _, r := range res {
		v, err := semver.NewVersion(r.Chart.Version)
		if err != nil || constraint.Check(v) {
			data = append(data, r)
		}
	}

	return data, nil

}

func buildIndex(settings *helm_env.EnvSettings) (*search.Index, error) {
	// Load the repositories.yaml
	rf, err := repo.LoadRepositoriesFile(settings.Home.RepositoryFile())
	if err != nil {
		return nil, err
	}

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		f := settings.Home.CacheIndex(n)
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			log.Warn("WARNING: Repo %q is corrupt or missing. Try 'helm repo update'.", n)
			continue
		}

		i.AddRepo(n, ind, true)
	}
	return i, nil
}
