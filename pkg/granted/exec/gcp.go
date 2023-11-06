package exec

import (
	"fmt"
	"strings"
	"syscall"

	osexec "os/exec"

	accessv1alpha1 "github.com/common-fate/ciem/gen/commonfate/cloud/access/v1alpha1"
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

var gcpCommand = cli.Command{
	Name:  "gcp",
	Usage: "Execute a command against a particular GCP project",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "project"},
		&cli.StringFlag{Name: "role"},
	},
	ArgsUsage: "--project <project> --role <role> -- <command to execute>",
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

		clio.Debugf("found executable %s", argv0)

		argv := make([]string, 0, 1+len(args))
		argv = append(argv, command)
		argv = append(argv, args...)

		env := envAsMap()

		env["CLOUDSDK_CORE_PROJECT"] = project

		clio.Debugf("running: CLOUDSDK_CORE_PROJECT=%s %s %s", project, command, strings.Join(args, " "))

		return syscall.Exec(argv0, argv, env.StringSlice())
	},
}
