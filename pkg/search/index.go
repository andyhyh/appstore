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
