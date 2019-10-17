// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pod

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/defaults"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/keystore"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/pod"
	commonvolume "github.com/cloudptio/logstash-operator/pkg/controller/common/volume"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/name"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/volume"
	"github.com/cloudptio/logstash-operator/pkg/utils/stringsutil"

	corev1 "k8s.io/api/core/v1"
)

const (
	// MonitorHTTPPort is the (default) port used by Logstash
	MonitorHTTPPort                      = 9600
	BeatsHTTPPort                        = 5044
	defaultImageRepositoryAndName string = "docker.elastic.co/logstash/logstash-oss"
)

// ports to set in the Logstash container
var ports = []corev1.ContainerPort{
	{Name: "monitor", ContainerPort: int32(MonitorHTTPPort), Protocol: corev1.ProtocolTCP},
	{Name: "beats", ContainerPort: int32(BeatsHTTPPort), Protocol: corev1.ProtocolTCP},
}

var DefaultResources = corev1.ResourceRequirements{
	Requests: map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	},
	Limits: map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	},
}

func imageWithVersion(image string, version string) string {
	return stringsutil.Concat(image, ":", version)
}

func NewPodTemplateSpec(ls v1beta1.Logstash, keystore *keystore.Resources) corev1.PodTemplateSpec {

	esURL := ls.AssociationConf().GetURL()

	logstashPipelineVolume := commonvolume.NewConfigMapVolumeWithMode(
		name.PipelineConfigMap(ls.Name), volume.PipelineVolumeName, volume.PipelineVolumeMountPath, int32(volume.PipelineVolumeMode))

	builder := defaults.NewPodTemplateBuilder(ls.Spec.PodTemplate, v1beta1.LogstashContainerName).
		WithResources(DefaultResources)

	lsContainer := GetLogstashContainer(builder.PodTemplate.Spec)
	memReq := lsContainer.Resources.Requests.Memory().String()

	memReq = strings.ReplaceAll(memReq, "Gi", "g")
	memReq = strings.ReplaceAll(memReq, "Mi", "m")

	builder = builder.WithLabels(label.NewLabels(ls.Name)).
		WithDockerImage(ls.Spec.Image, imageWithVersion(defaultImageRepositoryAndName, ls.Spec.Version)).
		WithPorts(ports).
		WithVolumes(logstashPipelineVolume.Volume()).
		WithVolumeMounts(logstashPipelineVolume.VolumeMount()).
		WithEnv(
			corev1.EnvVar{
				Name:  "ELASTICSEARCH_HOST",
				Value: esURL,
			},
			// corev1.EnvVar{
			// 	Name:  "ELASTICSEARCH_PORT",
			// 	Value: "9200",
			// },
			corev1.EnvVar{
				Name:  "LS_JAVA_OPTS",
				Value: fmt.Sprintf("-Xmx%s -Xms%s", memReq, memReq),
			},

			corev1.EnvVar{
				Name:  "HTTP_HOST",
				Value: "0.0.0.0",
			},
			corev1.EnvVar{
				Name:  "HTTP_PORT",
				Value: "9600",
			},
			corev1.EnvVar{
				Name:  "CONFIG_RELOAD_AUTOMATIC",
				Value: "true",
			},
			corev1.EnvVar{
				Name:  "PATH_CONFIG",
				Value: volume.PipelineVolumeMountPath,
			},
			corev1.EnvVar{
				Name:  "PATH_DATA",
				Value: volume.DataVolumeMountPath,
			},
			corev1.EnvVar{
				Name:  "QUEUE_CHECKPOINT_WRITES",
				Value: "1",
			},
			corev1.EnvVar{
				Name:  "QUEUE_DRAIN",
				Value: "true",
			},
			corev1.EnvVar{
				Name:  "QUEUE_MAX_BYTES",
				Value: "1gb",
			},
			corev1.EnvVar{
				Name:  "QUEUE_TYPE",
				Value: "persisted",
			},
		)

	if keystore != nil {
		builder.WithVolumes(keystore.Volume).
			WithInitContainers(keystore.InitContainer).
			WithInitContainerDefaults()
	}

	return builder.PodTemplate
}

// GetLogstashContainer returns the Logstash container from the given podSpec.
func GetLogstashContainer(podSpec corev1.PodSpec) *corev1.Container {
	return pod.ContainerByName(podSpec, v1beta1.LogstashContainerName)
}
