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
* the content charts.yaml
* information about available versions


`GET /packages?query=wordpress`

Filter on a query string.



### Install an application

`POST /releases`

```
{
  "application": "wordpress",
  "version": "4.1",
  "namespace": "default",
  "adminGroups": [
    "fc:uninett:avd:system"
  ],
  "development": false,
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
  "application": "wordpress",
  "version": "4.1",
  "namespace": "default",
  "adminGroups": [
    "fc:uninett:avd:system"
  ],
  "development": false,
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
