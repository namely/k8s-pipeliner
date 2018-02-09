# Kubernetes Pipeliner

[![Build Status](https://travis-ci.org/namely/k8s-pipeliner.svg?branch=master)](https://travis-ci.org/namely/k8s-pipeliner)
[![Coverage Status](https://coveralls.io/repos/github/namely/k8s-pipeliner/badge.svg?branch=master)](https://coveralls.io/github/namely/k8s-configurator?branch=master)

This tool is used to generate pipeline JSON for [Spinnaker](https://spinnaker.io) from Kubernetes manifest files.

The basic premise is that Kubernetes already has a well defined standard for how to define cluster resources (ReplicaSets, Deployments, etc), but there's no way to glue them into Spinnaker pipeline stages. That's what this tool aims to provide.

Using a pipeline configuration YAML file, you can define stages and reference Kubernetes manifest definitions to help fill in:

* The environment variables
* The container image
* The command and arguments
* Ports and load balancers

## Example Pipeline

This is a lengthy example of how a pipeline YAML looks. Each stage can reference a notification plugin you've installed in Spinnaker.

```
name: Example Deployment
application: example     # should match application created in spinnaker
triggers:                # list of triggers (currently only jenkins is supported)
- jenkins:
    job: "Example/job/master"
    master: "jenkins"
    propertyFile: "build.properties"
stages:
- account: int-k8s
  name: "Migrate INT"
  refId: "1"
  runJob:
    manifestFile: test-deployment.yml
    container: # override default command defined in the manifest
      command:
        - bundle
        - exec
        - rake
        - db:migrate
  notifications:
    - address: "#launchpad"
      type: "slack"
      when:
        - stage.complete
        - stage.failed
      message:
        stage.complete: |
          The stage has completed!
        stage.failed: |
          The stage has failed!
- account: int-k8s
  name: "Deploy to INT"
  refId: "2"
  reliesOn:
    - "1"
  deploy:
    manifestFile: test-deployment.yml
    maxRemainingASGS: 2 # total amount of replica sets to keep around afterwards before deleting
    scaleDown: true
    stack: web # primarily for labeling purposes on the created resources
    strategy: redblack
    targetSize: 2 # this is the total amount of replicas for the deployment
- account: int-k8s
  name: "Proceed to Staging?"
  refId: "3"
  reliesOn:
    - "2"
  manualJudgement:
    failPipeline: true
    instructions: |
      If this stage has completed QA, press proceed.
```

### Deployment Annotations

Right now this tool only supports rendering from Kubernetes Deployment manifests, this will likely change in the future but it works for right now.

To populate the `imageDescription` field that Spinnaker uses when deploying server clusters, this tool relies on annotations defined on your manifest:

```
apiVersion: extensions/v1beta2
kind: Deployment
metadata:
  name: example
  namespace: production
  annotations:
    namely.com/spinnaker-image-description-account: "your-registry"
    namely.com/spinnaker-image-description-imageid: "${ trigger.properties['docker_image'] }"
    namely.com/spinnaker-image-description-registry: "your.registry.land"
    namely.com/spinnaker-image-description-repository: "org/example"
    namely.com/spinnaker-image-description-organization: "namely"
    namely.com/spinnaker-image-description-tag: "${ trigger.properties['docker_tag'] }"
    namely.com/spinnaker-load-balancers: "example"
```

To attach load balancers to the resulting server group, you can add the annotation `namely.com/spinnaker-load-balancers` with a comma separated list of load balancers you've added to Spinnaker to attach them upon deployment.

## Installation

If you have a Go environment installed and configured, you can use `go get` to install the latest package of this project:

```
$ go get -u github.com/namely/k8s-pipeliner/cmd/k8s-pipeliner
```

To use it:

```
$ cd your-project
$ k8s-pipeliner create pipeline.yml
```

If you want a pretty view and have JQ installed, you can do:

```
$ k8s-pipeliner create pipeline.yml | jq .
```

To copy the result to your clipboard and you're on a Mac, you can do:

```
$ k8s-pipeliner create pipeline.yml | pbcopy
```

### Adding the Pipeline JSON

Once you've copied the resulting JSON from the pipeline configuration, you can go modify an already created Pipeline by clicking "Pipeline Actions" -> "Edit as JSON".

![](https://i.imgur.com/LoTrkBP.png)

Paste the JSON, and then in the bottom right of the screen click "Save".


## Schema

Here are the independent pieces of schema for pipeline.yml that you can use

### Manual Judgement

If you want to have a manual judgement in your pipeline, you can define a `manualJudgement` step within the `stages` array:

```
stages:
- name: "Continue Deploy?"
  manualJudgement:
    failPipeline: true
    instructions: |
      Once you're confident with this deploy, please approve it to continue.
```

* If `failPipeline` is set to true, the manual judgement must be approved for the rest of the pipeline to continue.
* The `instructions` are displayed within the UI when the pipeline is stalled waiting for a manual judgement. This is useful for whoever is providing the manual judgement to have context.

### Run A Job

A Job is a step in a pipeline that runs a one off task. A good example might be running a database migration before rolling out a piece of code.

```
stages:
- name: "Run Migrations"
  runJob:
    manifestFile: manifests/deployment.yaml
    container:
      command:
        - bundle
        - exec
      args:
       - rake
       - db:migrate
```

* `manifestFile` is used to generate the majority of the stage JSON for running the container. Things like environment, volumes, commands, etc are all stored within this Kubernetes Manifest file.
* `container` key is used to overwrite some of the values that are provided in the `manifestFile`. For example, if you want to run a migration script that is provided in the container instead of the default `rails server`, this is where you'd define it.
* `command` portion of the container override overwrites the `command` portion of the container being run in the job.
* `args` portion of the container override overwrites the `args` portion of the container being run in the job.

### Deploying Groups

A Deploy stage is used for running new server groups. You can use this stage to deploy several groups in-tandem. This is useful if you're deploying the same container for different application needs. IE: One is a consumer and another is a publisher.

```
stages:
- name: "Deploy"
  deploy:
    groups:
    - manifestFile: manifests/deployment.yaml
      maxRemainingASGS: 2
      scaleDown: true
      stack: web
      details: genpop
      strategy: redblack
      targetSize: 10
      loadBalancers:
        - namely
```

* `manifestFile` is used to generate the majority of the stage JSON for running the container. Things like environment, volumes, commands, etc are all stored within this Kubernetes Manifest file.
* `maxRemainingASGS` determines how many ReplicaSets Spinnaker will keep around after a deploy occurs. If using `redblack` strategy you need at least 2. This is used for rolling back deploys.
* `scaleDown` scales down the previous server group after a deploy. If you want traffic to be routed to both deployments set this to `false`
* `stack` is concatenated to the application name when deploying. So `application-stack` would be a result. CANNOT have dashes.
* `details` is concatenated to the application name and stack when deploying. So `application-stack-detail` would be a result. This can have multiple dashes.
* `strategy` is used to determine which strategy Spinnaker should use when deploying this new group.
* `targetSize` is the amount of replicas to be deployed to the Kubernetes cluster. This is _not_ taken from the `deployment` manifest file
* `loadBalancers` are the Spinnaker load balancers to be attached to this deployment. An array of strings. These will need to be defined inside of Spinnaker before a deploy to work.
