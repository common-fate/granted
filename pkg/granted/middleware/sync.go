package middleware

import (
	"github.com/common-fate/granted/pkg/autosync"
	"github.com/urfave/cli/v2"
)

func WithAutosync() cli.BeforeFunc {
	return func(ctx *cli.Context) error {
		autosync.Run()
		return nil
	}
}
