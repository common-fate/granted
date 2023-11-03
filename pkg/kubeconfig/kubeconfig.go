package kubeconfig

type Config struct {
	APIVersion     string    `yaml:"apiVersion"`
	Clusters       []Cluster `yaml:"clusters"`
	Contexts       []Context `yaml:"contexts"`
	CurrentContext string    `yaml:"current-context"`
	Kind           string    `yaml:"kind"`
	Users          []User    `yaml:"users"`
}

type Cluster struct {
	Cluster ClusterInfo `yaml:"cluster"`
	Name    string      `yaml:"name"`
}

type ClusterInfo struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	Server                   string `yaml:"server"`
}

type Context struct {
	Context ContextInfo `yaml:"context"`
	Name    string      `yaml:"name"`
}

type ContextInfo struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type User struct {
	Name string   `yaml:"name"`
	User UserInfo `yaml:"user"`
}

type UserInfo struct {
	Exec ExecInfo `yaml:"exec"`
}

type ExecInfo struct {
	APIVersion         string `yaml:"apiVersion"`
	Command            string `yaml:"command"`
	InstallHint        string `yaml:"installHint"`
	ProvideClusterInfo bool   `yaml:"provideClusterInfo"`
}
