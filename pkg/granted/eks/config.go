package eks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/granted/kubeconfig"
	"github.com/common-fate/granted/pkg/granted/proxy"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/fatih/color"
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

func AddContextToConfig(ensureAccessOutput *proxy.EnsureAccessOutput[*accessv1alpha1.AWSEKSProxyOutput], port string) error {

	kc, err := OpenKubeConfig()
	if err != nil {
		return err
	}

	clusterContextName := fmt.Sprintf("cf-grant-to-%s-as-%s", ensureAccessOutput.GrantOutput.EksCluster.Name, ensureAccessOutput.GrantOutput.ServiceAccountName)
	// Use the same name for the context and the cluster, so that each grant is assigned a unique entry for the cluster
	clusterName := clusterContextName

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
			Server:                fmt.Sprintf("http://localhost:%s", port),
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
			// @TODO, teams may wish to specify a default namespace for each user or cluster?
			Namespace: "default",
			Cluster:   clusterName,
			User:      username,
		},
	})
	if err != nil {
		return err
	}

	//add users
	err = kc.AddUser(&kubeconfig.UserConfig{
		Name: username,
		User: kubeconfig.AuthInfo{
			As: username,
		},
	})
	if err != nil {
		return err
	}

	//set the context
	clio.Infof("EKS proxy is ready for connections")
	clio.Infof("Your `~/.kube/config` file has been updated with a new cluster context. To connect to this cluster, run the following command to switch your current context:")
	clio.Log(color.YellowString("kubectl config use-context %s", clusterContextName))
	clio.NewLine()
	clio.Infof("Or using the --context flag with kubectl: %s", color.YellowString("kubectl --context=%s", clusterContextName))
	clio.NewLine()
	err = kc.SaveConfig()
	if err != nil {
		return err
	}
	return nil

}
