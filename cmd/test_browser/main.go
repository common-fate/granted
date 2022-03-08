package main

import (
	"fmt"
	"os"
)

// This package is used to test browser features in tests
func main() {
	shellEnv := os.Getenv("SHELL")
	fmt.Printf("shellEnv: %v\n", shellEnv)
	shellEnvo := os.Getenv("SHELLO")
	fmt.Printf("shellEnvo: %v\n", shellEnvo)
	os.Exit(0)
}
