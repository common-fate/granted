// Copyright 2023 Volvo Car Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/common-fate/clio"
)

const (
	version = "v1"
	kind    = "Config"
)

// New creates a new kubeconfig.
func New() *Config {
	return &Config{
		APIVersion: version,
		Kind:       kind,
		Clusters:   []*ClusterConfig{},
		Users:      []*UserConfig{},
		Contexts:   []*ContextConfig{},
	}
}

// Config is a kubeconfig.
type Config struct {
	Kind           string           `json:"kind"`
	APIVersion     string           `json:"apiVersion"`
	Preferences    Preferences      `json:"preferences"`
	CurrentContext string           `json:"current-context"`
	Clusters       []*ClusterConfig `json:"clusters"`
	Contexts       []*ContextConfig `json:"contexts"`
	Users          []*UserConfig    `json:"users"`
}

// UserConfig is a user in a kubeconfig.
type UserConfig struct {
	Name string   `json:"name"`
	User AuthInfo `json:"user"`
}

// ContextConfig is a context in a kubeconfig.
type ContextConfig struct {
	Name    string  `json:"name"`
	Context Context `json:"context"`
}

// ClusterConfig is a cluster in a kubeconfig.
type ClusterConfig struct {
	Name    string  `json:"name"`
	Cluster Cluster `json:"cluster"`
}

// GetCluster returns if the cluster with provided name is already present.
func (c *Config) SaveConfig() error {
	kubeConfigBytes, err := c.Marshal()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	kubeConfigPath := filepath.Join(homeDir, ".kube", "config")

	err = os.WriteFile(kubeConfigPath, kubeConfigBytes, 0700)
	if err != nil {
		return err
	}
	return nil
}

// AddCluster adds a cluster to the kubeconfig.
func (c *Config) AddCluster(cluster *ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("add cluster: %w", errIsNil)
	}
	if reflect.ValueOf(cluster.Cluster).IsZero() {
		return fmt.Errorf("add cluster: %w", errIsEmpty)
	}

	c.Clusters = append(c.Clusters, cluster)
	return nil
}

// AddUser adds a user to the kubeconfig.
func (c *Config) AddUser(user *UserConfig) error {
	if user == nil {
		return fmt.Errorf("add user: %w", errIsNil)
	}
	if reflect.ValueOf(user.User).IsZero() {
		return fmt.Errorf("add user: %w", errIsEmpty)
	}
	c.Users = append(c.Users, user)
	return nil
}

// AddContext adds a context to the kubeconfig.
func (c *Config) AddContext(context *ContextConfig) error {
	if context == nil {
		return fmt.Errorf("add context: %w", errIsNil)
	}
	if reflect.ValueOf(context.Context).IsZero() {
		return fmt.Errorf("add context: %w", errIsEmpty)
	}

	c.Contexts = append(c.Contexts, context)
	return nil
}

// GetCluster returns if the cluster with provided name is already present.
func (c *Config) GetCluster(name string) *ClusterConfig {
	for _, c := range c.Clusters {
		if c.Name == name {
			return c
		}
	}

	return nil
}

// ContextExists returns true if the context with provided name exists
func (c *Config) ContextExists(name string) bool {
	for _, c := range c.Contexts {
		if c.Name == name {
			return true
		}
	}

	return false
}

// UpdateCurrentContext updates the current context
func (c *Config) UpdateCurrentContext(name string) error {
	exists := c.ContextExists(name)

	if !exists {
		return fmt.Errorf("context not found: %s", name)
	}

	c.CurrentContext = name

	err := c.SaveConfig()
	if err != nil {
		return err
	}

	clio.Success("updated context")
	return nil
}
