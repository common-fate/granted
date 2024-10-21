// Copyright (c) 2023 Volvo Car Corporation
// SPDX-License-Identifier: Apache-2.0

/*
Package kubeconfig provides a simple way to manipulate kubeconfig files.

It allows you to :

  - [Load] a kubeconfig file from disk
  - [Merge] multiple kubeconfig files

Note that it doesn't support to [Write] a kubeconfig file to disk
as this can be done by [config.Marshal]ing the config and writing it to disk.
*/
package kubeconfig
