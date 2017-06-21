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

package status

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/uninett/appstore/pkg/helmutil"
	helm_env "k8s.io/helm/pkg/helm/environment"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
)

func GetAllReleases(settings *helm_env.EnvSettings, logger *logrus.Entry) ([]*release.Release, error) {
	client := helmutil.InitHelmClient(settings)
	sortBy := services.ListSort_NAME
	sortOrder := services.ListSort_ASC

	res, err := client.ListReleases(
		helm.ReleaseListLimit(256),
		helm.ReleaseListOffset(""),
		helm.ReleaseListFilter(""),
		helm.ReleaseListSort(int32(sortBy)),
		helm.ReleaseListOrder(int32(sortOrder)),
		helm.ReleaseListStatuses(statusCodes()),
		helm.ReleaseListNamespace(""),
	)

	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}

	if len(res.Releases) == 0 {
		return make([]*release.Release, 0), nil
	}

	if res.Next != "" {
		logger.Debug("\tnext: %s\n", res.Next)
	}

	return res.Releases, nil
}

// statusCodes gets the list of status codes that are to be included in the results.
func statusCodes() []release.Status_Code {
	return []release.Status_Code{
		release.Status_UNKNOWN,
		release.Status_DEPLOYED,
		release.Status_DELETING,
		release.Status_FAILED,
	}
}
