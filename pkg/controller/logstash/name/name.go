// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package name

import (
	common_name "github.com/cloudptio/logstash-operator/pkg/controller/common/name"
)

const (
	httpServiceSuffix       = "http"
	pipelineConfigMapSuffix = "pipeline"
)

// LSNamer is a Namer that is configured with the defaults for resources related to a Logstash resource.
var LSNamer = common_name.NewNamer("ls")

func HTTPService(lsName string) string {
	return LSNamer.Suffix(lsName, httpServiceSuffix)
}

func Deployment(lsName string) string {
	return LSNamer.Suffix(lsName)
}

func PipelineConfigMap(lsName string) string {
	return LSNamer.Suffix(lsName, pipelineConfigMapSuffix)
}
