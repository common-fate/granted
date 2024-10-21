// Copyright 2023 Volvo Car Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeconfig

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

// Unmarshal unmarshals a kubeconfig file from a byte slice.
func (c *Config) Unmarshal(b []byte) error {
	b, err := yaml.YAMLToJSON(b)
	if err != nil {
		return fmt.Errorf("unmarshall: convert yaml to json: %w", err)
	}
	err = json.Unmarshal(b, c)
	if err != nil {
		return fmt.Errorf("unmarshall json: %w", err)
	}
	return nil
}

// Marshal marshals a kubeconfig file to a byte slice.
func (c *Config) Marshal() ([]byte, error) {
	j, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshall: %w", err)
	}
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		return nil, err
	}
	return y, nil
}
