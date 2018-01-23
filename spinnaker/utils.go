package spinnaker

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	v1beta1 "k8s.io/api/apps/v1beta1"
)

type DeployStageConfig struct {
	Account       string
	Application   string
	DockerAccount string
	TagFormat     string
}

// DeployStageFromK8sDep creates a spinnaker deploy stage (clusters) from a
// manifest definition.
// NOTE: OLD dont use this
// NOTE: OLD dont use this
// NOTE: OLD dont use this
// NOTE: OLD dont use this
func DeployStageFromK8sDep(cfg DeployStageConfig, dep *v1beta1.Deployment) DeployStage {
	var containers []DeployStageContainer
	for _, container := range dep.Spec.Template.Spec.Containers {
		args := []string{}
		if container.Args != nil {
			args = container.Args
		}

		u, err := url.Parse(fmt.Sprintf("http://%s", container.Image))
		if err != nil {
			logrus.WithError(err).Fatal("could not parse image")
		}

		repository := strings.Split(u.Path, ":")[0][1:]

		c := DeployStageContainer{
			Args:            args,
			Command:         container.Command,
			ImagePullPolicy: "ALWAYS",
			Name:            container.Name,
			VolumeMounts:    []interface{}{},
			ImageDescription: ImageDescription{
				Account:     cfg.DockerAccount,
				FromTrigger: true,
				ImageID:     fmt.Sprintf("%s/%s:%s", u.Host, repository, cfg.TagFormat),
				Registry:    u.Hostname(),
				Repository:  repository,
				Tag:         cfg.TagFormat,
			},
		}

		for _, port := range container.Ports {
			c.Ports = append(c.Ports, struct {
				ContainerPort int32  `json:"containerPort"`
				Name          string `json:"name"`
				Protocol      string `json:"protocol"`
			}{port.ContainerPort, port.Name, string(port.Protocol)})
		}

		for _, env := range container.Env {
			var e EnvVar
			e.Name = env.Name
			e.Value = env.Value

			if vf := env.ValueFrom; vf != nil {
				if vf.ConfigMapKeyRef != nil {
					e.EnvSource = &EnvSource{
						ConfigMapSource: &ConfigMapSource{
							ConfigMapName: vf.ConfigMapKeyRef.Name,
							Key:           vf.ConfigMapKeyRef.Key,
						},
					}
				}

				if vf.SecretKeyRef != nil {
					e.EnvSource = &EnvSource{
						SecretSource: &SecretSource{
							Key:        vf.SecretKeyRef.Key,
							SecretName: vf.SecretKeyRef.Name,
						},
					}
				}
			}

			c.EnvVars = append(c.EnvVars, e)
		}

		c.Requests.CPU = container.Resources.Requests.Cpu().String()
		c.Requests.Memory = container.Resources.Requests.Memory().String()

		c.Limits.CPU = container.Resources.Limits.Cpu().String()
		c.Limits.Memory = container.Resources.Limits.Memory().String()

		containers = append(containers, c)
	}

	return DeployStage{
		Type: "deploy",
		Name: fmt.Sprintf("Deploy %s", dep.Name),
		Clusters: []Cluster{
			{
				Account:                        cfg.Account,
				Application:                    cfg.Application,
				Containers:                     containers,
				Region:                         "production",
				Namespace:                      "production",
				CloudProvider:                  "kubernetes",
				DNSPolicy:                      "ClusterFirst",
				Events:                         []interface{}{},
				Provider:                       "kubernetes",
				Strategy:                       "redblack",
				TargetSize:                     int(*dep.Spec.Replicas),
				VolumeSources:                  []interface{}{},
				ScaleDown:                      true,
				InterestingHealthProviderNames: []string{"KubernetesContainer", "KubernetesPod"},
				LoadBalancers:                  []string{}, // TODO: fix
				MaxRemainingAsgs:               3,
			},
		},
	}
}
