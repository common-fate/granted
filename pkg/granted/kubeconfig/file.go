// Copyright 2023 Volvo Car Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeconfig

import (
	"fmt"
	"os"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Load loads a kubeconfig file from the given path.
func Load(path string, validate ...ValidationFunc) (*Config, error) {
	if !fileExists(path) {
		return nil, fmt.Errorf("load kubeconfig: file %q does not exist", path)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"load kubeconfig: failed to read file %q: %w",
			path,
			err,
		)
	}

	c := New()

	if err = c.Unmarshal(b); err != nil {
		return nil, fmt.Errorf(
			"load kubeconfig: config %q invalid: %w",
			path,
			err,
		)
	}
	if err = checkNotEmpty(c); err != nil {
		return nil, fmt.Errorf(
			"load kubeconfig: config %q invalid: %w",
			path,
			err,
		)
	}

	if len(validate) > 0 {
		for _, v := range validate {
			if verr := v(c); verr != nil {
				return nil, fmt.Errorf(
					"load kubeconfig: config %q invalid: %w",
					path,
					verr,
				)
			}
		}
	}

	sortConfigEntries(c)

	return c, nil
}

func checkNotEmpty(c *Config) error {
	if cmp.Equal(c, New(), cmpopts.EquateEmpty()) {
		return fmt.Errorf("config %w", errIsEmpty)
	}
	if err := atLeastOneEntry(c); err != nil {
		return err
	}
	return nil
}

func sortConfigEntries(c *Config) {
	sort.SliceStable(
		c.Clusters, func(i, j int) bool {
			return c.Clusters[i].Name < c.Clusters[j].Name
		},
	)
	sort.SliceStable(
		c.Users, func(i, j int) bool {
			return c.Users[i].Name < c.Users[j].Name
		},
	)
	sort.SliceStable(
		c.Contexts, func(i, j int) bool {
			return c.Contexts[i].Name < c.Contexts[j].Name
		},
	)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
