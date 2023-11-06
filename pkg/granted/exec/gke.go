package exec

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	osexec "os/exec"

	"github.com/bufbuild/connect-go"
	"github.com/common-fate/ciem/config"
	accessv1alpha1 "github.com/common-fate/ciem/gen/commonfate/cloud/access/v1alpha1"
	attestv1alpha1 "github.com/common-fate/ciem/gen/commonfate/cloud/attest/v1alpha1"
	"github.com/common-fate/ciem/service/access"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/kubeconfig"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/container/v1"
	"gopkg.in/yaml.v3"
)

var gkeCommand = cli.Command{
	Name:  "gke",
	Usage: "Execute a command against a particular GKE cluster",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "project"},
		&cli.StringFlag{Name: "cluster"},
		&cli.StringFlag{Name: "location"},
		&cli.StringFlag{Name: "role"},
	},
	ArgsUsage: "--project <project> --cluster <cluster> --location <location> --role <role> -- <command to execute>",
	Action: func(c *cli.Context) error {
		ctx := c.Context

		clio.Debugf("exec command %s", strings.Join(c.Args().Slice(), " "))

		command := c.Args().First()
		args := c.Args().Tail()

		// bail out early if the command doesn't exist
		argv0, err := osexec.LookPath(command)
		if err != nil {
			return fmt.Errorf("couldn't find the executable '%s': %w", command, err)
		}

		project := c.String("project")
		resource := &accessv1alpha1.Resource{
			Resource: &accessv1alpha1.Resource_GcpProject{
				GcpProject: &accessv1alpha1.GCPProject{
					Project: project,
					Role:    c.String("role"),
				},
			},
		}

		err = assertAccess(ctx, resource)
		if err != nil {
			clio.Errorf("Error while ensuring access to GCP: %s\nContinuing anyway but you may receive an Access Denied error from the cloud provider...", err)
		}

		containersvc, err := container.NewService(ctx)
		if err != nil {
			return err
		}

		clusterName := c.String("cluster")
		location := c.String("location")

		cluster, err := containersvc.Projects.Locations.Clusters.Get(path.Join("projects", project, "locations", location, "clusters", clusterName)).Do()
		if err != nil {
			return err
		}

		kubeconfigfile, err := os.CreateTemp("", "kubeconfig*")
		if err != nil {
			return err
		}

		defer kubeconfigfile.Close()

		defer func() {
			err := os.Remove(kubeconfigfile.Name())
			if err != nil {
				clio.Errorf("error removing temporary kubeconfig file %s: %s", kubeconfigfile.Name(), err)
			}
		}()

		kc := kubeconfig.GenerateGKE(project, location, clusterName, cluster)

		err = yaml.NewEncoder(kubeconfigfile).Encode(kc)
		if err != nil {
			return err
		}

		clio.Debugf("wrote kubeconfig to %s", kubeconfigfile.Name())
		clio.Debugf("found executable %s", argv0)

		argv := make([]string, 0, 1+len(args))
		argv = append(argv, command)
		argv = append(argv, args...)

		env := envAsMap()

		env["KUBECONFIG"] = kubeconfigfile.Name()

		clio.Debugf("running: KUBECONFIG=%s %s %s", kubeconfigfile.Name(), command, strings.Join(args, " "))

		return syscall.Exec(argv0, argv, env.StringSlice())
	},
}

func assertAccess(ctx context.Context, resource *accessv1alpha1.Resource) error {
	cfg, err := config.LoadDefault(ctx)
	if err != nil {
		// just log a debug message
		clio.Debugf("could not load Common Fate context: ", err)
		return nil
	}

	client := access.NewFromConfig(cfg)

	ent, err := client.GetEntitlement(ctx, connect.NewRequest(&accessv1alpha1.GetEntitlementRequest{
		Resource: resource,
	}))
	if err != nil {
		return err
	}

	// TODO: add check for provisioning state - if it's in the provisioning state we want to poll until it's ready
	if ent.Msg.Entitlement.Status != accessv1alpha1.EntitlementStatus_ENTITLEMENT_STATUS_ACTIVE {
		switch r := ent.Msg.Entitlement.Resource.Resource.(type) {
		case *accessv1alpha1.Resource_GcpProject:
			clio.Infof("provisioning access to %s with role %s...", r.GcpProject.Project, r.GcpProject.Role)
		}

		res, err := client.CreateAccessRequest(ctx, connect.NewRequest(&accessv1alpha1.CreateAccessRequestRequest{
			Resources: []*accessv1alpha1.Resource{
				resource,
			},
			Justification: &accessv1alpha1.Justification{
				DeviceAttestation: &attestv1alpha1.Attestation{
					Header: &attestv1alpha1.Header{
						Version:       1,
						Timestamp:     time.Now().UnixMilli(),
						Type:          attestv1alpha1.AttestationType_ATTESTATION_TYPE_ACCESS_REQUEST,
						ContentDigest: []byte{}, // TODO
						Kid:           "ARP-guOO9ZhuI7GNrs9e7_qwrsoT4TF4w-LkL1NLlSbLAA",
					},
					Signature: []byte{}, // TODO
				},
			},
		}))
		if err != nil {
			return err
		}

		for _, e := range res.Msg.AccessRequest.Entitlements {
			gcp := e.Resource.GetGcpProject()
			if gcp == nil {
				continue
			}

			if e.Status == accessv1alpha1.EntitlementStatus_ENTITLEMENT_STATUS_ACTIVE {
				clio.Successf("access to %s with role %s is now active", gcp.Project, gcp.Role)
			}
		}
	} else {
		clio.Debugw("skipping requesting access", "status", ent.Msg.Entitlement.Status)
	}
	return nil
}

type EnvMap map[string]string

func (e EnvMap) StringSlice() []string {
	var res []string
	for k, v := range e {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return res
}

func envAsMap() EnvMap {
	m := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			m[e[:i]] = e[i+1:]
		}
	}

	return m
}
