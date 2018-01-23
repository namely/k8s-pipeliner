package types

// SpinnakerPipeline defines the fields for the top leve object of a spinnaker
// pipeline. Mostly used for constructing JSON
type SpinnakerPipeline struct {
	Triggers []Trigger `json:"triggers"`
}

// Trigger is an interface to encompass multiple types of Spinnaker triggers
type Trigger interface {
	spinnakerTrigger()
}

// JenkinsTrigger constructs the JSON necessary to include a Jenkins trigger
// for a spinnaker pipeline
type JenkinsTrigger struct {
	Enabled      bool   `json:"enabled"`
	Job          string `json:"job"`
	Master       string `json:"master"`
	PropertyFile string `json:"propertyFile"`
	Type         string `json:"type"`
}

var _ Trigger = &JenkinsTrigger{}

// Trigger implements Trigger
func (t *JenkinsTrigger) spinnakerTrigger() {}

// Stage is an interface to represent a Stage struct such as RunJob or Deploy
type Stage interface {
	spinnakerStage()
}

// RunJobStage is used to represent a job that needs to be ran in a pipeline
// configuration
type RunJobStage struct {
	Account              string            `json:"account"`
	Annotations          map[string]string `json:"annotations"`
	Application          string            `json:"application"`
	CloudProvider        string            `json:"cloudProvider"`
	CloudProviderType    string            `json:"cloudProviderType"`
	Container            Container         `json:"container"`
	DNSPolicy            string            `json:"dnsPolicy"`
	Labels               map[string]string `json:"labels"`
	Name                 string            `json:"name"`
	Namespace            string            `json:"namespace"`
	RefID                string            `json:"refId"`
	RequisiteStageRefIds []string          `json:"requisiteStageRefIds"`
	Type                 string            `json:"type"`
	VolumeSources        []interface{}     `json:"volumeSources"`
}

func (rjs RunJobStage) spinnakerStage() {}

var _ Stage = RunJobStage{}

// DeployStage handles the creation of a server cluster in a pipeline
type DeployStage struct {
	Clusters             []Cluster `json:"clusters"`
	Name                 string    `json:"name"`
	RefID                string    `json:"refId,omitempty"`
	RequisiteStageRefIds []string  `json:"requisiteStageRefIds,omitempty"`
	Type                 string    `json:"type"`
}

func (ds DeployStage) spinnakerStage() {}

var _ Stage = DeployStage{}

// Cluster defines a server group to be deployed within a Deploy stage of a
// pipeline
type Cluster struct {
	Account                        string            `json:"account"`
	Application                    string            `json:"application"`
	CloudProvider                  string            `json:"cloudProvider"`
	Containers                     []Container       `json:"containers"`
	DNSPolicy                      string            `json:"dnsPolicy"`
	Events                         []interface{}     `json:"events"`
	FreeFormDetails                string            `json:"freeFormDetails"`
	InterestingHealthProviderNames []string          `json:"interestingHealthProviderNames"`
	LoadBalancers                  []string          `json:"loadBalancers"`
	MaxRemainingAsgs               int               `json:"maxRemainingAsgs"`
	Namespace                      string            `json:"namespace"`
	NodeSelector                   map[string]string `json:"nodeSelector"`
	PodAnnotations                 map[string]string `json:"podAnnotations"`
	Provider                       string            `json:"provider"`

	// Region is just a kubernetes namespace
	Region                        string            `json:"region"`
	ReplicaSetAnnotations         map[string]string `json:"replicaSetAnnotations"`
	ScaleDown                     bool              `json:"scaleDown"`
	SecurityGroups                []interface{}     `json:"securityGroups"`
	Stack                         string            `json:"stack"`
	Strategy                      string            `json:"strategy"`
	TargetSize                    int               `json:"targetSize"`
	TerminationGracePeriodSeconds int               `json:"terminationGracePeriodSeconds"`
	VolumeSources                 []interface{}     `json:"volumeSources"`
	DelayBeforeDisableSec         int               `json:"delayBeforeDisableSec,omitempty"`
}

// ManualJudgementStage handles the creation of a manual judgement
type ManualJudgementStage struct {
	FailPipeline         bool          `json:"failPipeline"`
	Instructions         string        `json:"instructions"`
	JudgmentInputs       []interface{} `json:"judgmentInputs"`
	Name                 string        `json:"name"`
	Notifications        []interface{} `json:"notifications"`
	RefID                string        `json:"refId,omitempty"`
	RequisiteStageRefIds []string      `json:"requisiteStageRefIds,omitempty"`
	Type                 string        `json:"type"`
}

func (mjs ManualJudgementStage) spinnakerStage() {}

var _ Stage = ManualJudgementStage{}

// Container is a representation of a container to be deployed either as a job
// or within a cluster
type Container struct {
	Args             []string         `json:"args"`
	Command          []string         `json:"command"`
	EnvVars          []EnvVar         `json:"envVars"`
	ImageDescription ImageDescription `json:"imageDescription"`
	ImagePullPolicy  string           `json:"imagePullPolicy"`
	Limits           Resources        `json:"limits"`
	Requests         Resources        `json:"requests"`

	Name  string `json:"name"`
	Ports []Port `json:"ports"`

	VolumeMounts []interface{} `json:"volumeMounts"`
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
}

// ConfigMapSource is a env var from a config map in k8s
type ConfigMapSource struct {
	ConfigMapName string `json:"configMapName"`
	Key           string `json:"key"`
}

// ImageDescription is used to tell spinnaker which image to use for a stage
// of a pipeline in a cluster definition or job
type ImageDescription struct {
	Account     string `json:"account"`
	FromTrigger bool   `json:"fromTrigger"`
	ImageID     string `json:"imageId"`
	Registry    string `json:"registry"`
	Repository  string `json:"repository"`
	Tag         string `json:"tag"`
}
