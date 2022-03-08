package main

import (
	"context"
	"fmt"
	"os"

	"github.com/common-fate/granted/pkg/cfaws"
)

func main() {
	if !cfaws.GetEnvCredentials(context.Background()).HasKeys() {
		fmt.Println("No credentials set in env")
		os.Exit(1)
	}
}
