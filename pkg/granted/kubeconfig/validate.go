// Copyright 2023 Volvo Car Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeconfig

import (
	"errors"
	"fmt"
)

var (
	errIsNil   = errors.New("is nil")
	errIsEmpty = errors.New("is empty")
)

// ValidationFunc is used to validate a kubeconfig.
type ValidationFunc func(*Config) error

// WithValidContexts checks that each context has a valid cluster and user.
func WithValidContexts(c *Config) error {
	clusterSet := make(map[string]struct{})
	userSet := make(map[string]struct{})

	for _, cluster := range c.Clusters {
		clusterSet[cluster.Name] = struct{}{}
	}
	for _, user := range c.Users {
		userSet[user.Name] = struct{}{}
	}
	for _, ctx := range c.Contexts {
		// check that each context has a valid cluster and user
		if _, ok := clusterSet[ctx.Context.Cluster]; !ok {
			return fmt.Errorf(
				"context %q references unknown cluster %q",
				ctx.Name,
				ctx.Context.Cluster,
			)
		}
		if _, ok := userSet[ctx.Context.User]; !ok {
			return fmt.Errorf(
				"context %q references unknown user %q",
				ctx.Name,
				ctx.Context.User,
			)
		}
	}
	return nil
}

func atLeastOneEntry(c *Config) error {
	if c == nil {
		return fmt.Errorf("config %w", errIsNil)
	}
	var ee []error
	if len(c.Clusters) == 0 {
		ee = append(ee, fmt.Errorf("no clusters defined"))
	}
	if len(c.Users) == 0 {
		ee = append(ee, fmt.Errorf("no users defined"))
	}
	if len(c.Contexts) == 0 {
		ee = append(ee, fmt.Errorf("no contexts defined"))
	}
	if len(ee) > 0 {
		var err error
		for i, e := range ee {
			if i == 0 {
				err = e
			} else {
				err = fmt.Errorf("%v + %w", e, err)
			}
		}
		return err
	}
	return nil
}
