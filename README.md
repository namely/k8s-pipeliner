# Kubernetes Pipeliner

[![Build Status](https://travis-ci.org/namely/k8s-pipeliner.svg?branch=master)](https://travis-ci.org/namely/k8s-pipeliner)
[![Coverage Status](https://coveralls.io/repos/github/namely/k8s-pipeliner/badge.svg)](https://coveralls.io/github/namely/k8s-pipeliner)

This tool is used to generate pipeline JSON for [Spinnaker](https://spinnaker.io) from Kubernetes manifest files.

The basic premise is that Kubernetes already has a well defined standard for how to define cluster resources (ReplicaSets, Deployments, etc), but there's no way to glue them into Spinnaker pipeline stages. That's what this tool aims to provide.

Using a pipeline configuration YAML file, you can define stages and reference Kubernetes manifest definitions to help fill in:

* The environment variables
* The container image
* The command and arguments
* Ports and load balancers

## Example Pipeline

This is a lengthy example of how a pipeline YAML looks.

```yaml
name: Example Deployment
application: example

disableConcurrentExecutions: true
keepQueuedPipelines: true
description: This pipeline deploys some sweet code

notifications:
  - address: "#launchpad"
    type: "slack"
    when:
      - pipeline.complete
      - pipeline.failed
    message:
      pipeline.complete: |
        The stage has completed!
      pipeline.failed: |
        The stage has failed!

parameters:
  - name: "random"
    description: "random description"
    required: true
    default: "hello-world"

triggers:
- jenkins:
    job: "Example/job/master"
    master: "jenkins"
    propertyFile: "build.properties"
    enabled: true
- webhook:
    source: "random-string"
    enabled: true
imageDescriptions:
  - name: main-image
    account: "namely-registry"
    image_id: "${ trigger.properties['docker_image'] }"
    registry: "registry.namely.land"
    repository: "namely/example-all-day"
    tag: "${ trigger.properties['docker_tag'] }"
    organization: "namely"
  - name: second-image
    account: "namely-registry"
    image_id: "${ trigger.properties['docker_image'] }"
    registry: "registry.namely.land"
    repository: "namely/second-repo-name"
    tag: "${ trigger.properties['docker_tag'] }"
    organization: "namely"
stages:
- account: int-k8s
  name: "Migrate INT"
  runJob:
    manifestFile: test-deployment.yml
    imageDescriptions:
      - name: main-image
        containerName: example
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
  deploy:
    groups:
      - manifestFile: test-deployment.yml
        maxRemainingASGS: 2 # total amount of replica sets to keep around afterwards before deleting
        scaleDown: true
        stack: web # primarily for labeling purposes on the created resources
        details: helloworld
        strategy: redblack
        targetSize: 2 # this is the total amount of replicas for the deployment
        containerOverrides: {}
        imageDescriptions:
          - name: main-image
            containerName: example
          - name: second-image
            containerName: init-example
        loadBalancers:
          - "test"
- account: int-k8s
  name: "Proceed to Staging?"
  refId: "3"
  reliesOn:
    - "2"
  manualJudgement:
    timeoutHours: 48
    failPipeline: true
    instructions: |
      If this stage has completed QA, press proceed.
```

## <a name="installation"></a> Installation

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
$ k8s-pipeliner create --linear pipeline.yml | pbcopy
```

### <a name="installation"></a> Upgrade k8s-pipeliner

Pull the latest from master branch and run
```
make install
```

### <a name="pipelinejson"></a> Adding the Pipeline JSON

Once you've copied the resulting JSON from the pipeline configuration, you can go modify an already created Pipeline by clicking "Pipeline Actions" -> "Edit as JSON".

![](https://i.imgur.com/LoTrkBP.png)

Paste the JSON, and then in the bottom right of the screen click "Save".


## <a name="schema"></a> Schema

Here are the independent pieces of schema for pipeline.yml that you can use. You can also take a look at the [Config Definitions](pipeline/config/config.go).

### <a name="triggers"></a> Triggers

We currently support 2 types of triggers in k8s-pipeliner, webhooks and jenkins.

#### <a name="webhooks"></a> Webhooks

```yaml
triggers:
- webhook:
    source: "random-string"
    enabled: true
```

The "source" field in webhooks is the endpoint that you need to hit in order to kick off Spinnaker. If you're running the gate API at "gate-api.example.com", your webhook endpoint from this configuration would be https://gate-api.example.com/webhooks/webhook/random-string (random-string is our source value)

#### <a name="jenkins"></a> Jenkins

Spinnaker also supports triggering off of Jenkins jobs completing, to use this trigger include a `jenkins` field:

```
triggers:
- jenkins:
    job: "Example/job/master"
    master: "jenkins"
    propertyFile: "build.properties" # optional
    enabled: true
```

### <a name="manualjudgement"></a> Manual Judgement

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

### <a name="jobs"></a> Run A Job

A Job is a step in a pipeline that runs a one off task. A good example might be running a database migration before rolling out a piece of code.

```
stages:
- name: "Run Migrations"
  runJob:
    manifestFile: manifests/deployment.yml
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

### <a name="groups"></a> Deploying Groups

A Deploy stage is used for running new server groups. You can use this stage to deploy several groups in-tandem. This is useful if you're deploying the same container for different application needs. IE: One is a consumer and another is a publisher.

```yaml
imageDescriptions:
  - name: main-image
    account: "namely-registry"
    image_id: "${ trigger.properties['docker_image'] }"
    registry: "registry.namely.land"
    repository: "namely/example-all-day"
    tag: "${ trigger.properties['docker_tag'] }"
    organization: "namely"

stages:
- name: "Deploy"
  deploy:
    groups:
    - manifestFile: manifests/deployment.yml
      maxRemainingASGS: 2
      scaleDown: true
      stack: web
      details: genpop
      strategy: redblack
      targetSize: 10
      loadBalancers:
        - namely
      imageDescriptions:
        - name: main-image
          containerName: my-container
```

* `manifestFile` is used to generate the majority of the stage JSON for running the container. Things like environment, volumes, commands, etc are all stored within this Kubernetes Manifest file.
* `maxRemainingASGS` determines how many ReplicaSets Spinnaker will keep around after a deploy occurs. If using `redblack` strategy you need at least 2. This is used for rolling back deploys.
* `scaleDown` scales down the previous server group after a deploy. If you want traffic to be routed to both deployments set this to `false`
* `stack` is concatenated to the application name when deploying. So `application-stack` would be a result. CANNOT have dashes.
* `details` is concatenated to the application name and stack when deploying. So `application-stack-detail` would be a result. This can have multiple dashes.
* `strategy` is used to determine which strategy Spinnaker should use when deploying this new group.
* `targetSize` is the amount of replicas to be deployed to the Kubernetes cluster. This is _not_ taken from the `deployment` manifest file
* `loadBalancers` are the Spinnaker load balancers to be attached to this deployment. An array of strings. These will need to be defined inside of Spinnaker before a deploy to work.

#### <a name="images"></a> Image Descriptions

`imageDescriptions` are a top level key in your YAML that define a Docker image to be deployed. You can use Spinnaker expression syntax in these fields to add dynamic images in your deployment pipeline.

The `name` key on the image description is then referenced in your `groups` of a `deploy` stage. You can see this here:

```yaml
imageDescriptions:
  - name: main-image
    containerName: my-container
```

What k8s-pipeliner does is it looks into the manifest you've supplied, finds the container with the name "my-container", and includes the image description for the Spinnaker JSON that is rendered for it. This allows you to specify multiple containers in your pods and be able to swap out the images based on dynamic values for them.

### <a name="parameters"></a> Parameter Support

This tool also supports the ability to include parameters in your pipeline definitions:

```yaml
parameters:
  - name: "random"
    description: "random description"
    required: true
    default: "hello-world"
```

This configures your pipeline to have parameters in the UI / enable pipeline expressions.

Files under the `configuratorFiles` section are expected to be in the [k8s-configurator format](https://github.com/namely/k8s-configurator/blob/master/README.md#input-file-and-envs). These will be run through k8s-configurator to generate the environment-specific manifest. By default, the environment used by k8s-configurator will be determined by the account used in this stage. However, you may set the optional `env` property for configuratorFiles to override this.

```yaml
stages:
- account: ops-k8s
  name: "Deploy"
  deployEmbeddedManifests:
    moniker:
      app: app
      cluster: cluster
      detail: detail
      stack: stack
    files:
      - file: manifests/deployment.yml
      - file: manifests/service.yml
      - file: manifests/migrate-job.yml
    configuratorFiles:
      - file: test-configurator.yml
        env: superOps
```

All of these files will be composed into a single stage deployment into the given account. This means you can deploy services and deployments in tandem together.
