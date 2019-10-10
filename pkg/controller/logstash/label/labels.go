// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package label

import "github.com/elastic/cloud-on-k8s/pkg/controller/common"

const (
	// LogstashNameLabelName used to represent a Logstash in k8s resources
	LogstashNameLabelName = "logstash.k8s.elastic.co/name"

	// Type represents the Logstash type
	Type = "logstash"
)

// NewLabels constructs a new set of labels for a Logstash pod
func NewLabels(logstashName string) map[string]string {
	return map[string]string{
		LogstashNameLabelName:  logstashName,
		common.TypeLabelName: Type,
	}
}
