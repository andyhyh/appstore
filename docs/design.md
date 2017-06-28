# App Store Design

[Overall K8s AppStore architecture](https://scm.uninett.no/researchlab/jupyter/blob/master/docs/appstore.md)


## App Store Backend API

All responses are JSON encoded.


### List and navigate the application library

All these endpoints are public, unprotected.


`GET /packages`

List all packages. A JSON list.

Each entry should contain info from repo.

Each pacakge should contain information about:

* the package ID
* the source repo
* the content `charts.yaml`
* information about available versions


`GET /packages?query=wordpress`

Filter on a query string.

`GET /packages?repo=researchlab`

Filter for a specifc repo.


### Get available namespaces


`GET /namespaces`

The user needs to be authenticated. Returns a list of namespaces that the enduser is allowed to deploy to.

For now, the namespaces setup could be implemented as a static configuration file at the backend, including a list of group to namespace mappings. This endpoint lists the configuration filtered with the ones that the authenticated user is allowed to based upon the groups the user is member of.

Response similar to this:

```
[
  {
    "id": "researchlab",
    "name": "Research Lab prosjektet",
    "groups": [
      "fc:orgunit:systemavdelingen", "fc:adhoc:bcca03b7-8193-4692-91e0-3c0715756a26"
    ]
  },
  {
    "id": "uninett-experimental",
    "name": "Experimental services",
    "groups": [
      "fc:orgunit:systemavdelingen"
    ]
  }
]
```


### Install an application

`POST /releases`

```
{
  "repo": "researchlab",        # OPTIONAL
  "package": "wordpress",       # REQUIRED
  "version": "4.1",             # OPTIONAL
  "namespace": "default",       # OPTIONAL
  "adminGroups": [              # OPTIONAL
    "fc:uninett:avd:system"
  ],
  "values": {
    "name": "A nice blog about kubernetes",
    "host": "k8s-blog.lab.uninett-apps.no"
  }
}
```

Install an application. The user needs to be authenticated.

The response is identical to the accepted values of the input, in addition to the ID and the owner.

200 OK implies a successful response from tiller.

```
{
  "id": "blurry-green-cat",
  "owner": "d7e71800-549b-40ad-ae5c-d88891327231",
  "repo": "researchlab",
  "package": "wordpress",
  "version": "4.1",
  "namespace": "default",
  "adminGroups": [
    "fc:uninett:avd:system"
  ],
  "values": {
    "name": "A nice blog about kubernetes",
    "host": "k8s-blog.lab.uninett-apps.no"
  }
}
```




`GET /releases/{blurry-green-cat}`

Authentication needed. The user needs to be the owner of the release or member in on of the adminGroup-s registered with the release.

Returns the same output as the response from installation as seen above.

`GET /releases/{blurry-green-cat}/status`

Returns status of the deployment similar to the output of `helm status {...}`:

```
{
  "lastDeployed": "2017-06-02 12:34:20",
  "namespace": "default",
  "status": "DEPLOYED",
  "resources": [
    {}, {}, {}
  ]
}
```

It would be nice to also get some additional metadata here about third party provisioned resources, like a Dataporten registration. It could be included like this:

```
{
  "dataporten": {
    "client_id": "a5472bb9-35a0-48e1-bf75-28e828e84df7"
  }
}
```




### List releases

`GET /releases`

Authentication needed. Filter releases to the ones the user is the owner of or member in on of the adminGroup-s registered with the release.



### Upgrading a release to a newer version of the application

`PATCH /releases/{blurry-green-cat}`

```
{
  "version": "4.2"
}
```

### Deleting an deployment


`DELETE /releases/{blurry-green-cat}`

Will delete the release.
