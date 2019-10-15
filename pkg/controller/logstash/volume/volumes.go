// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package volume

const (
	DataVolumeName      = "logstash-data"
	DataVolumeMountPath = "/usr/share/logstash/data"

	PipelineVolumeName      = "pipeline"
	PipelineVolumeMountPath = "/usr/share/logstash/pipeline"
	PipelineVolumeMode      = 420
)
