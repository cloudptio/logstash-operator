#!/usr/bin/env bash

# Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
# or more contributor license agreements. Licensed under the Elastic License;
# you may not use this file except in compliance with the Elastic License.

# Script to generate a NOTICE file containing licence information from dependencies.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR=${SCRIPT_DIR}/../..

build_licence_detector() {
    (
        cd $SCRIPT_DIR
        go build -v github.com/cloudptio/logstash-operator/hack/licence-detector
    )
}

generate_notice() {
    (
        cd $PROJECT_DIR
        go list -m -json all | ${SCRIPT_DIR}/licence-detector -template=${SCRIPT_DIR}/NOTICE.txt.tmpl -out=${PROJECT_DIR}/NOTICE.txt
    )
}

build_licence_detector
generate_notice
