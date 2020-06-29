<!-- PROJECT SHIELDS -->
<!--
*** I'm using markdown "reference style" links for readability.
*** Reference links are enclosed in brackets [ ] instead of parentheses ( ).
*** See the bottom of this document for the declaration of the reference variables
*** for contributors-url, forks-url, etc. This is an optional, concise syntax you may use.
*** https://www.markdownguide.org/basic-syntax/#reference-style-links
-->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]
[![LinkedIn][linkedin-shield]][linkedin-url]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/mainak90/helmer">
    <img src="logo.png" alt="Logo" width="80" height="80">
  </a>

  <h3 align="center">Helmer: A lightweight helm repo!</h3>

  <p align="center">
    An awesome README template to jumpstart your projects!
    <br />
    <a href="https://github.com/mainak90/helmer"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/mainak90/helmer">View Demo</a>
    ·
    <a href="https://github.com/mainak90/helmer">Report Bug</a>
    ·
    <a href="https://github.com/mainak90/helmer">Request Feature</a>
  </p>
</p>

<!-- ABOUT THE PROJECT -->
## Components

The following are the important components of the application

Here's why:
* The view is basically the index.html. This doubles as the index page for multipart uploads of helm charts,
* Datamodels are in models/
* Controllers to upload charts, list them, remove them, deploy charts, list releases and delete chart releases are in here.
* The controllers use helper functions on utils/.
* The associated helm chart is on chart/


### Built With

* [Golang](https://golang.org/)

### Endpoints

```
    "/uploadChart" : Uploads the chart into the filesystem 
    Method: POST
```

```
    "/getChartList": Fetches the list of helm charts uploaded into the filesystem
    Method: GET
```

```
    "/deployChart": Deploy the chart into the local or remote kubernetes cluster
    Method: POST
```

```
    "/getDeploymentList": List the helm releases already deployed into the cluster and their status.
    Method: GET
```

```
    "/deleteDeployment/namespace/{namespace}/name/{name}": Delete/Uninstall the helm release from cluster.
    Method: POST
```

### Quick start

#### Using helm

Go to the charts directory, make sure helm is installed.

```
    helm package helmer
    helm install helmer helmer-0.1.0.tgz --namespace <namespace>
    For now this installation uses a NodePort service, you need to use <NodeIP>:<NodePort> to access the services.
```

#### Run locally

```
    go run main.go
```
