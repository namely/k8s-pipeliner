package builder_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
)

type EmbeddedManifestTest struct {
	suite.Suite

	pipeline *config.Pipeline
}

func (em *EmbeddedManifestTest) BeforeTest(suiteName, testName string) {
	em.pipeline = &config.Pipeline{
		Stages: []config.Stage{},
	}
}

func (em *EmbeddedManifestTest) AppendStage(stage config.Stage) {
	em.pipeline.Stages = append(em.pipeline.Stages, stage)
}

func (em *EmbeddedManifestTest) Builder() *builder.Builder {
	return builder.New(em.pipeline)
}

func (em *EmbeddedManifestTest) TestFilesAreBuilt() {
	em.AppendStage(config.Stage{
		Name: "deploy nginx",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			Files: []config.ManifestFile{
				{
					File: "testdata/nginx-deployment.yml",
				},
			},
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.ManifestStage)
	em.Require().True(ok)
	em.Equal("deploy nginx", stg.Name)

	em.Require().Len(stg.Manifests, 1)

	deploy, ok := stg.Manifests[0].(*appsv1.Deployment)
	em.Require().True(ok)
	em.Equal("nginx-deployment", deploy.GetName())
}

func (em *EmbeddedManifestTest) TestMonikerAnnotationsAreIncluded() {
	em.AppendStage(config.Stage{
		Name: "deploy nginx",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			Files: []config.ManifestFile{
				{
					File:    "testdata/nginx-deployment-annotations.yml",
					Moniker: &config.Moniker{App: "fake-app", Cluster: "fake-cluster", Stack: "fake-stack", Detail: "fake-details"},
				},
			},
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.ManifestStage)
	em.Require().True(ok)
	em.Equal("deploy nginx", stg.Name)

	em.Require().Len(stg.Manifests, 1)

	deploy, ok := stg.Manifests[0].(*appsv1.Deployment)
	em.Require().True(ok)
	em.Equal("fake-app", deploy.GetAnnotations()["moniker.spinnaker.io/application"])
	em.Equal("fake-cluster", deploy.GetAnnotations()["moniker.spinnaker.io/cluster"])
	em.Equal("fake-stack", deploy.GetAnnotations()["moniker.spinnaker.io/stack"])
	em.Equal("fake-detail", deploy.GetAnnotations()["moniker.spinnaker.io/detail"])
}

func TestEmbeddedManifests(t *testing.T) {
	em := &EmbeddedManifestTest{}
	suite.Run(t, em)
}
