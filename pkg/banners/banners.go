package banners

import (
	"fmt"

	"github.com/common-fate/granted/internal/build"
)

func WithVersion() string {
	return fmt.Sprintf("Granted version: %s\n", build.Version)
}
