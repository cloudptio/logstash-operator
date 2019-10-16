// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package volume

import (
	"github.com/cloudptio/logstash-operator/pkg/controller/common/volume"
)

const (
	DataVolumeName      = "logstash-data"
	DataVolumeMountPath = "/usr/share/kibana/data"

	PatternsVolumeName = "patterns"
	FilesVolumeName    = "files"
	PipelineVolumeName = "pipeline"

	PatternsVolumeMountPath = ""
	FilesVolumeMountPath = ""
	PipelineVolumeMountPath = "/usr/share/logstash/pipeline"

	PatternsVolumeConfigMapName = "logstash-patterns"
	FilesVolumeConfigMapName    = "logstash-files"
	PipelineVolumeConfigMapName = "logstash-pipeline"

	PatternsVolumeMode = 420
	FilesVolumeMode    = 420
	PipelineVolumeMode = 420
)

var LogstashPatternsVolume = volume.NewConfigMapVolumeWithMode(PatternsVolumeConfigMapName, PatternsVolumeName, PatternsVolumeMountPath, int32(PatternsVolumeMode))
var LogstashFilesVolume = volume.NewConfigMapVolumeWithMode(FilesVolumeConfigMapName, FilesVolumeName, FilesVolumeMountPath, int32(FilesVolumeMode))
var LogstashPipelineVolume = volume.NewConfigMapVolumeWithMode(PipelineVolumeConfigMapName, PipelineVolumeName, PipelineVolumeMountPath, int32(PipelineVolumeMode))
