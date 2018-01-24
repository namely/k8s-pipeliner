package builder_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainersFromManifests(t *testing.T) {
	wd, _ := os.Getwd()

	t.Run("Deployment manifests are returned correctly", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.full.yml")
		group, err := builder.ContainersFromManifest(file)

		require.NoError(t, err, "error on retrieving the deployment manifests")

		assert.Len(t, group.Containers, 1)
		assert.Len(t, group.Annotations, 2)
		assert.Equal(t, "fake-namespace", group.Namespace)
	})

	t.Run("Deployments schemes are converted to latest", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.v1beta1.yml")
		group, err := builder.ContainersFromManifest(file)

		require.NoError(t, err, "error on retrieving the deployment manifests")

		assert.Len(t, group.Containers, 1)
		assert.Len(t, group.Annotations, 2)
		assert.Equal(t, "fake-namespace", group.Namespace)
	})
}
