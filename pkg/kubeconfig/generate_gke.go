package kubeconfig

import (
	"fmt"

	"google.golang.org/api/container/v1"
)

func GenerateGKE(project string, location string, clusterName string, cluster *container.Cluster) Config {
	fullClusterName := fmt.Sprintf("gke_%s_%s_%s", project, location, clusterName)

	config := Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{
				Name: fullClusterName,
				Cluster: ClusterInfo{
					CertificateAuthorityData: cluster.MasterAuth.ClusterCaCertificate,
					Server:                   "https://" + cluster.Endpoint,
				},
			},
		},
		Contexts: []Context{
			{
				Context: ContextInfo{
					Cluster: fullClusterName,
					User:    fullClusterName,
				},
				Name: fullClusterName,
			},
		},
		CurrentContext: fullClusterName,
		Users: []User{
			{
				Name: fullClusterName,
				User: UserInfo{
					Exec: ExecInfo{
						APIVersion:         "client.authentication.k8s.io/v1beta1",
						Command:            "gke-gcloud-auth-plugin",
						InstallHint:        "Install gke-gcloud-auth-plugin for use with kubectl by following\nhttps://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke",
						ProvideClusterInfo: true,
					},
				},
			},
		},
	}

	return config
}
