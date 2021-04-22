package builder_test

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/suite"
	v1beta12 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
)

type EmbeddedManifestTest struct {
	suite.Suite
	kubecostData map[string][]byte
	pipeline     *config.Pipeline
}

func (em *EmbeddedManifestTest) BeforeTest(suiteName, testName string) {
	em.pipeline = &config.Pipeline{
		Stages: []config.Stage{},
	}
	kubecostResponse, _ := ioutil.ReadFile("kubecost_response_example_test.text")
	em.kubecostData = map[string][]byte{
		"development": kubecostResponse,
		"production":  kubecostResponse,
	}
}

func (em *EmbeddedManifestTest) AppendStage(stage config.Stage) {
	em.pipeline.Stages = append(em.pipeline.Stages, stage)
}

func (em *EmbeddedManifestTest) Builder() *builder.Builder {
	return builder.New(em.pipeline, builder.WithKubecostData(em.kubecostData))
}

func (em *EmbeddedManifestTest) BuilderWithBasePath(basePath string) *builder.Builder {
	return builder.New(em.pipeline, builder.WithBasePath(basePath), builder.WithKubecostData(em.kubecostData))
}

func (em *EmbeddedManifestTest) TestFilesAreBuilt() {
	em.AppendStage(config.Stage{
		Name:    "deploy nginx",
		Account: "int-k8s",
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

	deploy, ok := stg.Manifests[0].(*v1beta12.Deployment)
	em.Require().True(ok)
	em.Equal("nginx-deployment", deploy.GetName())
	em.Equal("nginx:1.7.9", deploy.Spec.Template.Spec.Containers[0].Image)
}

func (em *EmbeddedManifestTest) TestConfiguratorFilesNoEnv() {
	em.AppendStage(config.Stage{
		Name:    "deploy cm",
		Account: "int-k8s",
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
		Name:    "deploy cm",
		Account: "int-k8s",
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

func (em *EmbeddedManifestTest) TestConfiguratorFilesBasePath() {
	em.AppendStage(config.Stage{
		Name:    "deploy cm",
		Account: "int-k8s",
		DeployEmbeddedManifests: &config.DeployEmbeddedManifests{
			ConfiguratorFiles: []config.ManifestFile{
				{
					File:        "configurator.yml",
					Environment: "superOps",
				},
			},
		},
	})

	pipeline, err := em.BuilderWithBasePath("testdata").Pipeline()
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
		Name:    "deploy cm",
		Account: "int-k8s",
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
		Name:    "deploy nginx",
		Account: "int-k8s",
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
		Name:    "deploy nginx",
		Account: "int-k8s",
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
		Name:    "deploy nginx",
		Account: "int-k8s",
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

	_, dok := stg.Manifests[0].(*v1beta12.Deployment)
	em.Require().True(dok)
}

func (em *EmbeddedManifestTest) TestDeployEmbeddedManifestDefaultProperties() {
	boolt := true
	boolf := false

	em.AppendStage(config.Stage{
		Name:    "deploy nginx",
		Account: "int-k8s",
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

	deploy, ok := stg.Manifests[0].(*v1beta12.Deployment)
	em.Require().True(ok)
	em.Equal("nginx-deployment", deploy.GetName())
	em.Equal("nginx:1.7.9", deploy.Spec.Template.Spec.Containers[0].Image)

	em.Equal(&boolf, stg.CompleteOtherBranchesThenFail)
	em.Equal(&boolf, stg.ContinuePipeline)
	em.Equal(&boolt, stg.FailPipeline)
	em.Equal(&boolf, stg.MarkUnstableAsSuccessful)
	em.Equal(&boolt, stg.WaitForCompletion)
}

func (em *EmbeddedManifestTest) TestDeleteEmbeddedManifest() {
	boolt := true
	boolf := false
	em.AppendStage(config.Stage{
		Name:    "delete nginx",
		Account: "int-k8s",
		DeleteEmbeddedManifest: &config.DeleteEmbeddedManifest{
			File:                          "testdata/nginx-deployment.yml",
			CompleteOtherBranchesThenFail: &boolt,
			ContinuePipeline:              &boolt,
			FailPipeline:                  &boolf,
			MarkUnstableAsSuccessful:      &boolt,
			WaitForCompletion:             &boolf,
		},
	})

	pipeline, err := em.Builder().Pipeline()
	em.Require().NoError(err, "error building pipeline config")

	stg, ok := pipeline.Stages[0].(*types.DeleteManifestStage)
	em.Require().True(ok, "was not a delete manifest stage")
	em.Equal("delete nginx", stg.Name)
	em.Equal("Deployment nginx-deployment", stg.ManifestName)

	em.Equal(&boolt, stg.CompleteOtherBranchesThenFail)
	em.Equal(&boolt, stg.ContinuePipeline)
	em.Equal(&boolf, stg.FailPipeline)
	em.Equal(&boolt, stg.MarkUnstableAsSuccessful)
	em.Equal(&boolf, stg.WaitForCompletion)
}
func (em *EmbeddedManifestTest) TestDeleteEmbeddedManifestDefaultProperties() {
	boolt := true
	boolf := false
	em.AppendStage(config.Stage{
		Name:    "delete nginx",
		Account: "int-k8s",
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

	em.Equal(&boolf, stg.CompleteOtherBranchesThenFail)
	em.Equal(&boolf, stg.ContinuePipeline)
	em.Equal(&boolt, stg.FailPipeline)
	em.Equal(&boolf, stg.MarkUnstableAsSuccessful)
	em.Equal(&boolt, stg.WaitForCompletion)
}

func TestEmbeddedManifests(t *testing.T) {
	em := &EmbeddedManifestTest{}
	suite.Run(t, em)
}
