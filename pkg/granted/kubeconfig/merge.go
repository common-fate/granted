// Copyright 2023 Volvo Car Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeconfig

import (
	"errors"
	"fmt"

	"github.com/imdario/mergo"
)

// Merge merges multiple kubeconfig files into one.
// Note that the order of the files is important, as the first file will be
// the base for the merge, and the last file will be the final result.
func Merge(cc ...*Config) (*Config, error) {
	if len(cc) == 0 {
		return nil, errors.New("no config to merge")
	}

	r := New()
	for _, c := range cc {
		if err := checkNotEmpty(c); err != nil {
			return nil, fmt.Errorf("merge: %w", err)
		}
		if err := mergo.Merge(r, c, mergo.WithAppendSlice); err != nil {
			return nil, fmt.Errorf("merge: %w", err)
		}
	}

	sortConfigEntries(r)

	return r, nil
}
