module github.com/namely/k8s-pipeliner

go 1.12

require (
	github.com/hashicorp/go-multierror v1.0.0
	github.com/namely/k8s-configurator v0.0.4
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.22.4
	golang.org/x/net v0.0.0-20200425230154-ff2c4b7c35a0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.5
	k8s.io/client-go v11.0.0+incompatible
)
