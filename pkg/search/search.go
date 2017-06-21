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
	"github.com/Sirupsen/logrus"
	"github.com/uninett/appstore/pkg/debug"
	"time"

	"k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
)

func GetAllCharts(settings *helm_env.EnvSettings, logger *logrus.Entry) ([]*search.Result, error) {
	err := ensureIndex(settings, logger)
	if err != nil {
		return nil, err
	}

	res := index.All()
	return res, nil
}

func FindCharts(settings *helm_env.EnvSettings, query string, version string, logger *logrus.Entry) ([]*search.Result, error) {
	t1 := time.Now()
	err := ensureIndex(settings, logger)
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

	defer debug.GetFunctionTiming(t1,
		"search.SearchCharts returned",
		logrus.Fields{
			"query":       query,
			"num_results": len(data),
		},
		logger,
	)

	return data, err
}

func GetSinglePackage(settings *helm_env.EnvSettings, packageName string, logger *logrus.Entry) ([]*search.Result, error) {
	err := ensureIndex(settings, logger)
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
		logger.Debug(fmt.Sprintf("package %s not found", packageName))
		return nil, fmt.Errorf("package not found")
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
