package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/namely/k8s-pipeliner/pipeline/config"
)

func TestNewConfig(t *testing.T) {
	wd, _ := os.Getwd()
	file, err := os.Open(filepath.Join(wd, "testdata", "pipeline.full.yaml"))
	require.Nil(t, err, "error opening testdata file")

	cfg, err := config.NewPipeline(file)
	require.Nil(t, err, "error generating new config from file reader")

	assert.Equal(t, "Connect Deployment", cfg.Name)
	require.Len(t, cfg.Triggers, 1)
	require.Equal(t, "Connect/job/master", cfg.Triggers[0].Jenkins.Job)
	assert.Len(t, cfg.Stages, 2)
}
