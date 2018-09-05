package builder_test

import (
	"github.com/stretchr/testify/suite"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
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

	em.Equal("nginx test", pipeline.Name)
}
