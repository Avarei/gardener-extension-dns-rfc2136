#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-FileCopyrightText: 2024 metal-stack Authors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# We need to explicitly pass GO111MODULE=off to k8s.io/code-generator as it is significantly slower otherwise,
# see https://github.com/kubernetes/code-generator/issues/100.
export GO111MODULE=off

rm -f $GOPATH/bin/*-gen

PROJECT_ROOT=$(dirname $0)/..

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/avarei/gardener-extension-dns-rfc2136/pkg/client/componentconfig \
  github.com/avarei/gardener-extension-dns-rfc2136/pkg/apis \
  github.com/avarei/gardener-extension-dns-rfc2136/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/avarei/gardener-extension-dns-rfc2136/pkg/client/componentconfig \
  github.com/avarei/gardener-extension-dns-rfc2136/pkg/apis \
  github.com/avarei/gardener-extension-dns-rfc2136/pkg/apis \
  "config:v1alpha1" \
  --extra-peer-dirs=github.com/avarei/gardener-extension-dns-rfc2136/pkg/apis/config,github.com/avarei/gardener-extension-dns-rfc2136/pkg/apis/config/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${PROJECT_ROOT}/hack/LICENSE_BOILERPLATE.txt"
