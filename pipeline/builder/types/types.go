package types

import (
	corev1 "k8s.io/api/core/v1"
)

// SpinnakerPipeline defines the fields for the top leve object of a spinnaker
// pipeline. Mostly used for constructing JSON
type SpinnakerPipeline struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Application string `json:"application,omitempty"`

	Triggers      []Trigger      `json:"triggers"`
	Stages        []Stage        `json:"stages"`
	Notifications []Notification `json:"notifications"`

	// Pipeline level config
	LimitConcurrent      bool   `json:"limitConcurrent"`
	KeepWaitingPipelines bool   `json:"keepWaitingPipelines"`
	Description          string `json:"description"`

	Parameters []Parameter `json:"parameterConfig"`
}

// Parameter is a parameter declaration for a pipeline config
type Parameter struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Required    bool   `json:"required"`

	// TODO(bobbytables): Allow configuring parameter options
	HasOptions bool          `json:"hasOptions"`
	Options    []interface{} `json:"options"`
}

// Stage is an interface to represent a Stage struct such as RunJob or Deploy
type Stage interface {
	spinnakerStage()
}

// RunJobStage is used to represent a job that needs to be ran in a pipeline
// configuration
type RunJobStage struct {
	StageMetadata

	Account           string            `json:"account"`
	Annotations       map[string]string `json:"annotations"`
	Application       string            `json:"application"`
	CloudProvider     string            `json:"cloudProvider"`
	CloudProviderType string            `json:"cloudProviderType"`
	Container         *Container        `json:"container,omitempty"`
	DNSPolicy         string            `json:"dnsPolicy"`
	Labels            map[string]string `json:"labels,omitempty"`
	Namespace         string            `json:"namespace"`
	VolumeSources     []*VolumeSource   `json:"volumeSources,omitempty"`
}

func (rjs RunJobStage) spinnakerStage() {}

var _ Stage = RunJobStage{}

// DeployStage handles the creation of a server cluster in a pipeline
type DeployStage struct {
	StageMetadata

	Clusters []Cluster `json:"clusters"`
}

func (ds DeployStage) spinnakerStage() {}

var _ Stage = DeployStage{}

// ManualJudgementStage handles the manual judgement json in a pipeline
type ManualJudgementStage struct {
	StageMetadata

	FailPipeline bool     `json:"failPipeline"`
	Instructions string   `json:"instructions"`
	Inputs       []string `json:"inputs,omitempty"`
}

func (mjs ManualJudgementStage) spinnakerStage() {}

var _ Stage = ManualJudgementStage{}

// Cluster defines a server group to be deployed within a Deploy stage of a
// pipeline
type Cluster struct {
	Account                        string            `json:"account"`
	Application                    string            `json:"application"`
	CloudProvider                  string            `json:"cloudProvider"`
	Containers                     []*Container      `json:"containers"`
	DNSPolicy                      string            `json:"dnsPolicy"`
	Events                         []interface{}     `json:"events"`
	InitContainers                 []*Container      `json:"initContainers"`
	InterestingHealthProviderNames []string          `json:"interestingHealthProviderNames"`
	LoadBalancers                  []string          `json:"loadBalancers"`
	MaxRemainingAsgs               int               `json:"maxRemainingAsgs"`
	NodeSelector                   map[string]string `json:"nodeSelector,omitempty"`
	PodAnnotations                 map[string]string `json:"podAnnotations,omitempty"`
	Provider                       string            `json:"provider"`

	// Region is just a kubernetes namespace
	Region    string `json:"region"`
	Namespace string `json:"namespace"`

	ReplicaSetAnnotations         map[string]string `json:"replicaSetAnnotations,omitempty"`
	ScaleDown                     bool              `json:"scaleDown"`
	SecurityGroups                []interface{}     `json:"securityGroups,omitempty"`
	Stack                         string            `json:"stack"`
	Details                       string            `json:"freeFormDetails"`
	Strategy                      string            `json:"strategy"`
	TargetSize                    int               `json:"targetSize"`
	TerminationGracePeriodSeconds int               `json:"terminationGracePeriodSeconds"`
	VolumeSources                 []*VolumeSource   `json:"volumeSources,omitempty"`
	DelayBeforeDisableSec         int               `json:"delayBeforeDisableSec,omitempty"`
}

// Container is a representation of a container to be deployed either as a job
// or within a cluster
type Container struct {
	Args             []string         `json:"args,omitempty"`
	Command          []string         `json:"command,omitempty"`
	EnvVars          []EnvVar         `json:"envVars,omitempty"`
	EnvFrom          []EnvFromSource  `json:"envFrom,omitempty"`
	ImageDescription ImageDescription `json:"imageDescription"`
	ImagePullPolicy  string           `json:"imagePullPolicy"`
	Limits           Resources        `json:"limits"`
	Requests         Resources        `json:"requests"`

	Name  string `json:"name"`
	Ports []Port `json:"ports"`

	VolumeMounts []VolumeMount `json:"volumeMounts"`

	LivenessProbe  *Probe `json:"livenessProbe"`
	ReadinessProbe *Probe `json:"readinessProbe"`
}

// EnvFromSource is used to pull in a config map as a list of environment
// variables
type EnvFromSource struct {
	Prefix          string                  `json:"prefix"`
	ConfigMapSource *EnvFromConfigMapSource `json:"configMapRef,omitempty"`
	SecretSource    *EnvFromSecretSource    `json:"secretRef,omitempty"`
}

// EnvFromConfigMapSource is used to pull in a configmap for key/value envVars
type EnvFromConfigMapSource struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional"`
}

// EnvFromSecretSource is used to pull in a secret for key/value envVars
type EnvFromSecretSource struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional"`
}

// VolumeMount describes a mount that should be mounted in to the container
// by referencing a volume source in the pod spec
type VolumeMount struct {
	MountPath string `json:"mountPath"`
	Name      string `json:"name"`
	ReadOnly  bool   `json:"readOnly"`
}

// Resources for the container either as a limit or request
type Resources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// Port is a Container port to expose
type Port struct {
	ContainerPort int32  `json:"containerPort"`
	Name          string `json:"name"`
	Protocol      string `json:"protocol"`
}

// EnvVar represents an environment variable that is injected into the container
// when it starts. It can contain many sources
type EnvVar struct {
	EnvSource *EnvSource `json:"envSource,omitempty"`
	Name      string     `json:"name"`
	Value     string     `json:"value,omitempty"`
}

// EnvSource encapsulates the different possible places an environment var
// can come from (secrets or configmaps)
type EnvSource struct {
	SecretSource    *SecretSource    `json:"secretSource,omitempty"`
	ConfigMapSource *ConfigMapSource `json:"configMapSource,omitempty"`
}

// SecretSource is a env var from a secret map in k8s
type SecretSource struct {
	SecretName string `json:"secretName"`
	Key        string `json:"key"`
	Optional   bool   `json:"optional"`
}

// ConfigMapSource is a env var from a config map in k8s
type ConfigMapSource struct {
	ConfigMapName string `json:"configMapName"`
	Key           string `json:"key"`
	Optional      bool   `json:"optional"`
}

// ImageDescription is used to tell spinnaker which image to use for a stage
// of a pipeline in a cluster definition or job
type ImageDescription struct {
	Account      string `json:"account"`
	FromTrigger  bool   `json:"fromTrigger"`
	ImageID      string `json:"imageId"`
	Registry     string `json:"registry"`
	Repository   string `json:"repository"`
	Tag          string `json:"tag"`
	Organization string `json:"organization"`
}

// Notification is a struct for defining a notification for a stage or pipeline
type Notification struct {
	Address string                         `json:"address"`
	Level   string                         `json:"level"`
	Type    string                         `json:"type"`
	When    []string                       `json:"when"`
	Message map[string]NotificationMessage `json:"message"`
}

// NotificationMessage is for providing text to a stage failure type
type NotificationMessage struct {
	Text string `json:"text"`
}

// VolumeSource defines a pod volume source that can be referenced by containers
type VolumeSource struct {
	Name string `json:"name"`
	Type string `json:"type"`

	EmptyDir              *EmptyDirVolumeSource              `json:"emptyDir,omitempty"`
	ConfigMap             *ConfigMapVolumeSource             `json:"configMap,omitempty"`
	Secret                *SecretVolumeSource                `json:"secret,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
}

// EmptyDirVolumeSource defines a empty directory volume source for a pod:
// https://kubernetes.io/docs/api-reference/v1.9/#emptydirvolumesource-v1-core
type EmptyDirVolumeSource struct {
	Medium string `json:"medium"`
}

// ConfigMapVolumeSource type for referencing configmaps in volumes
type ConfigMapVolumeSource struct {
	ConfigMapName string             `json:"configMapName"`
	DefaultMode   *int32             `json:"defaultMode,omitempty"`
	Items         []corev1.KeyToPath `json:"items"`
}

// SecretVolumeSource for referencing secret types in volumes
type SecretVolumeSource struct {
	SecretName string             `json:"secretName"`
	Items      []corev1.KeyToPath `json:"items"`
}

// PersistentVolumeClaimVolumeSource for referencing secret types in volumes
type PersistentVolumeClaimVolumeSource struct {
	ClaimName string `json:"claimName"`
}

// Probe is a probe against a container for things such as liveness or readiness
type Probe struct {
	FailureThreshold    int32        `json:"failureThreshold"`
	InitialDelaySeconds int32        `json:"initialDelaySeconds"`
	PeriodSeconds       int32        `json:"periodSeconds"`
	SuccessThreshold    int32        `json:"successThreshold"`
	TimeoutSeconds      int32        `json:"timeoutSeconds"`
	Handler             ProbeHandler `json:"handler"`
}

// ProbeHandler represents all of the different types of probes
type ProbeHandler struct {
	ExecAction      *ExecAction      `json:"execAction,omitempty"`
	HTTPGetAction   *HTTPGetAction   `json:"httpGetAction,omitempty"`
	TCPSocketAction *TCPSocketAction `json:"tcpSocketAction,omitempty"`
	Type            string           `json:"type"`
}

// ExecAction is a probe type that runs a command
type ExecAction struct {
	Commands []string `json:"commands"`
}

// HTTPGetAction a probe type that hits an HTTP endpoint in the container
type HTTPGetAction struct {
	HTTPHeaders []HTTPGetActionHeaders `json:"httpHeaders"`
	Path        string                 `json:"path"`
	Port        int                    `json:"port"`
	URIScheme   string                 `json:"uriScheme"`
}

// HTTPGetActionHeaders is a key/value struct for headers used in a HTTP probe
type HTTPGetActionHeaders struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TCPSocketAction checks if a TCP connection can be opened against the given port
type TCPSocketAction struct {
	Port int `json:"port"`
}
