package builder_test

import (
	"testing"

	"github.com/kubernetes/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/stretchr/testify/suite"

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

	deploy, ok := stg.Manifests[0].(*unstructured.Unstructured)
	em.Require().True(ok)
	em.Equal("nginx-deployment", deploy.GetName())
	em.Equal("Deployment", deploy.GetKind())
}

func (em *EmbeddedManifestTest) TestConfiguratorFilesNoEnv() {
	em.AppendStage(config.Stage{
		Name: "deploy cm",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			ConfiguratorFiles: []config.ManifestFile{
				{
					File: "testdata/configurator.yml",
				},
			},
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.ManifestStage)
	em.Require().True(ok)
	em.Equal("deploy cm", stg.Name)

	em.Require().Len(stg.Manifests, 1)

	cm, ok := stg.Manifests[0].(*unstructured.Unstructured)
	em.Require().True(ok)
	em.Equal("configurator-test", cm.GetName())
	em.Equal("ConfigMap", cm.GetKind())
}

func (em *EmbeddedManifestTest) TestConfiguratorFiles() {
	em.AppendStage(config.Stage{
		Name: "deploy cm",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			ConfiguratorFiles: []config.ManifestFile{
				{
					File:        "testdata/configurator.yml",
					Environment: "superOps",
				},
			},
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.ManifestStage)
	em.Require().True(ok)
	em.Equal("deploy cm", stg.Name)

	em.Require().Len(stg.Manifests, 1)

	cm, ok := stg.Manifests[0].(*unstructured.Unstructured)
	em.Require().True(ok)
	em.Equal("configurator-test", cm.GetName())
	em.Equal("ConfigMap", cm.GetKind())
}

func (em *EmbeddedManifestTest) TestBadConfiguratorFiles() {
	em.AppendStage(config.Stage{
		Name: "deploy cm",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			ConfiguratorFiles: []config.ManifestFile{
				{
					File:        "testdata/bad-configurator.yml",
					Environment: "ops",
				},
			},
		},
	})

	_, err := em.Builder().Pipeline()
	em.Require().Error(err)
}

func (em *EmbeddedManifestTest) TestBadMultipleDocumentsError() {
	em.AppendStage(config.Stage{
		Name: "deploy nginx",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			DefaultMoniker: &config.Moniker{
				App:     "fake-app",
				Stack:   "fake-stack",
				Detail:  "fake-detail",
				Cluster: "fake-cluster",
			},
			Files: []config.ManifestFile{
				{
					File: "testdata/multiple-documents-bunk.yml",
				},
			},
		},
	})

	_, err := em.Builder().Pipeline()
	em.Require().Error(err)
}

func (em *EmbeddedManifestTest) TestMultipleDocumentsAreAdded() {
	em.AppendStage(config.Stage{
		Name: "deploy nginx",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			DefaultMoniker: &config.Moniker{
				App:     "fake-app",
				Stack:   "fake-stack",
				Detail:  "fake-detail",
				Cluster: "fake-cluster",
			},
			Files: []config.ManifestFile{
				{
					File: "testdata/multiple-documents.yml",
				},
			},
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.ManifestStage)
	em.Require().True(ok)
	em.Equal("deploy nginx", stg.Name)

	em.Require().Len(stg.Manifests, 3)

	dr, dok := stg.Manifests[2].(*unstructured.Unstructured)
	em.Require().True(dok)
	em.Equal("DestinationRule", dr.GetKind())
	em.Equal("networking.istio.io/v1alpha3", dr.GetAPIVersion())
}

func (em *EmbeddedManifestTest) TestMonikerAnnotationsAreIncluded() {
	em.AppendStage(config.Stage{
		Name: "deploy nginx",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			DefaultMoniker: &config.Moniker{
				App:     "fake-app",
				Stack:   "fake-stack",
				Detail:  "fake-detail",
				Cluster: "fake-cluster",
			},
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
	em.Equal("fake-app", stg.Moniker.App)
	em.Equal("fake-stack", stg.Moniker.Stack)
	em.Equal("fake-detail", stg.Moniker.Detail)
	em.Equal("fake-cluster", stg.Moniker.Cluster)

	_, dok := stg.Manifests[0].(*unstructured.Unstructured)
	em.Require().True(dok)
}

func (em *EmbeddedManifestTest) TestDeleteEmbeddedManifest() {
	em.AppendStage(config.Stage{
		Name: "delete nginx",
		DeleteEmbeddedManifest: &config.DeleteEmbeddedManifest{
			File: "testdata/nginx-deployment.yml",
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.DeleteManifestStage)
	em.Require().True(ok, "was not a delete manifest stage")
	em.Equal("delete nginx", stg.Name)
	em.Equal("Deployment nginx-deployment", stg.ManifestName)
}

func TestEmbeddedManifests(t *testing.T) {
	em := &EmbeddedManifestTest{}
	suite.Run(t, em)
}
