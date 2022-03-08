package main

import (
	"fmt"
	"os"
)

// This package is used to test browser features in tests
func main() {
	shellEnv := os.Getenv("SHELL")
	fmt.Printf("shellEnv: %v\n", shellEnv)
	os.Exit(0)
}
