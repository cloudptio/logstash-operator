// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package name

import (
	common_name "github.com/cloudptio/logstash-operator/pkg/controller/common/name"
)

const httpServiceSuffix = "http"

// LSNamer is a Namer that is configured with the defaults for resources related to a Kibana resource.
var KBNamer = common_name.NewNamer("kb")

func HTTPService(kbName string) string {
	return KBNamer.Suffix(kbName, httpServiceSuffix)
}

func Deployment(kbName string) string {
	return KBNamer.Suffix(kbName)
}
