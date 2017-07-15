package releaseutil

import "strings"

// The k8s resources returned by Tiller is a string with the format:
//
// RESOURCES:
// ==> v1/Secret
// NAME                TYPE    DATA  AGE
// ...
//
// ==> v1/PersistentVolumeClaim
// NAME                STATUS  VOLUME   CAPACITY  ACCESSMODES  STORAGECLASS  AGE
// ...
//
// ==> v1/Service
// NAME                CLUSTER-IP  EXTERNAL-IP  PORT(S)   AGE
// ...
//
// ==> v1beta1/Deployment
// NAME                DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
// excited-newt-mysql  1        1        1           1          8h
// ...
//
// which is inconvinent to pass to the user. We instead want to split it into a map like:
//
// "v1beta1/Deployment": {
//	[
//		{
//		"name": "excited-newt-mysql",
//		 "desired": 1, "current": 1, "up-to-date": 1,
//		 "age": 8h
//	  }
//	]
// }
//
// as this is easier to reuse.
func ParseResources(resourcesRaw string) map[string]map[string]string {
	resourcesSplit := strings.Split(resourcesRaw, "\n\n")

	parsedRes := make(map[string]map[string]string)
	for _, r := range resourcesSplit {
		lines := strings.Split(strings.TrimSpace(r), "\n")
		if len(lines) > 2 {
			// The title is of the format "==> $title-name"
			title := strings.TrimPrefix(lines[0], "==> ")
			col_names := strings.Fields(lines[1])
			for c_i, c_n := range col_names {
				col_names[c_i] = strings.ToLower(c_n)
			}

			items := make(map[string]string)
			for _, i := range lines[1:] {
				cols := strings.Fields(i)
				for c_i, c := range cols {
					items[col_names[c_i]] = c
				}
			}

			parsedRes[title] = items
		}
	}

	return parsedRes
}
