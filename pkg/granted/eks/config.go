package eks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/granted/kubeconfig"
)

func OpenKubeConfig() (*kubeconfig.Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	kubeConfigPath := filepath.Join(homeDir, ".kube", "config")

	kc, err := kubeconfig.Load(kubeConfigPath)
	if err != nil {
		clio.Errorf("error loading kubeconfig, proceeding trying to generate config anyway (%s)", err.Error())
	}

	if kc == nil {
		kc = kubeconfig.New()
	}

	return kc, nil
}

func AddContextToConfig(ensureAccessOutput *ensureAccessOutput) error {

	kc, err := OpenKubeConfig()
	if err != nil {
		return err
	}

	clusterName := fmt.Sprintf("cf-proxy-%s", ensureAccessOutput.GrantOutput.EksCluster.Name)
	clusterContextName := fmt.Sprintf("cf-grant-%s-%s", strings.ToLower(clusterName), ensureAccessOutput.GrantOutput.ServiceAccountName)
	username := ensureAccessOutput.GrantOutput.ServiceAccountName

	var contexts []*kubeconfig.ContextConfig
	for i, context := range kc.Contexts {
		if context.Name != clusterContextName {
			contexts = append(contexts, kc.Contexts[i])
		}
	}
	kc.Contexts = contexts

	// remove existing cluster definitions so they can be reset
	var clusters []*kubeconfig.ClusterConfig
	for i, cluster := range kc.Clusters {
		if cluster.Name != clusterName {
			clusters = append(clusters, kc.Clusters[i])
		}
	}
	kc.Clusters = clusters

	var users []*kubeconfig.UserConfig
	for i, user := range kc.Users {
		if user.Name != username {
			users = append(users, kc.Users[i])
		}
	}
	kc.Users = users

	//add the new cluster and context back in
	err = kc.AddCluster(&kubeconfig.ClusterConfig{
		Name: clusterName,
		Cluster: kubeconfig.Cluster{
			Server:                "http://localhost:5555", //todo update this to be the proxy url
			InsecureSkipTLSVerify: true,
		},
	})
	if err != nil {
		return err
	}

	//add the context back in
	err = kc.AddContext(&kubeconfig.ContextConfig{
		Name: clusterContextName,
		Context: kubeconfig.Context{
			Namespace: "default",
			Cluster:   clusterName,
			User:      username,
		},
	})
	if err != nil {
		return err
	}

	//add users
	// for _, u := range []string{"common-fate-readonly", "common-fate-admin"} {
	err = kc.AddUser(&kubeconfig.UserConfig{
		Name: username,
		User: kubeconfig.AuthInfo{
			Exec: &kubeconfig.ExecConfig{
				Command:         "cf",
				Args:            []string{"kube", "credentials"},
				InteractiveMode: kubeconfig.IfAvailableExecInteractiveMode,
				APIVersion:      "client.authentication.k8s.io/v1",
			},
			As: username,
		},
	})
	if err != nil {
		return err
	}

	// }

	//set the context
	clio.Warnf("`~/.kube/config` Updated. Set current context with: `kubectl config set-context %s`", clusterContextName)

	err = kc.SaveConfig()
	if err != nil {
		return err
	}
	return nil

}
