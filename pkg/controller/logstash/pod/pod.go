// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pod

import (
	"github.com/elastic/cloud-on-k8s/pkg/apis/logstash/v1beta1"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/defaults"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/keystore"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/pod"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/label"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/volume"
	"github.com/elastic/cloud-on-k8s/pkg/utils/stringsutil"

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

func imageWithVersion(image string, version string) string {
	return stringsutil.Concat(image, ":", version)
}

func NewPodTemplateSpec(ls v1beta1.Logstash, keystore *keystore.Resources) corev1.PodTemplateSpec {
	builder := defaults.NewPodTemplateBuilder(ls.Spec.PodTemplate, v1beta1.LogstashContainerName).
		WithLabels(label.NewLabels(ls.Name)).
		WithDockerImage(ls.Spec.Image, imageWithVersion(defaultImageRepositoryAndName, ls.Spec.Version)).
		WithPorts(ports).
		WithVolumes(volume.LogstashPipelineVolume.Volume()). //, volume.LogstashFilesVolume.Volume(), volume.LogstashPipelineVolume.Volume()).
		WithVolumeMounts(volume.LogstashPipelineVolume.VolumeMount()).
		WithEnv(
			corev1.EnvVar{
				Name: "HTTP_HOST",
				Value: "0.0.0.0",
			},
			corev1.EnvVar{
				Name: "HTTP_PORT",
				Value: "9600",
			},
			corev1.EnvVar{
				Name: "ELASTICSEARCH_HOST",
				Value: "quickstart-es-http.default",
			},
			corev1.EnvVar{
				Name: "ELASTICSEARCH_PORT",
				Value: "9200",
			},
			corev1.EnvVar{
				Name: "LS_JAVA_OPTS",
				Value: "-Xmx1g -Xms1g",
			},
			corev1.EnvVar{
				Name: "CONFIG_RELOAD_AUTOMATIC",
				Value: "true",
			},
			corev1.EnvVar{
				Name: "PATH_CONFIG",
				Value: "/usr/share/logstash/pipeline",
			},
			corev1.EnvVar{
				Name: "PATH_DATA",
				Value: "/usr/share/logstash/data",
			},
			corev1.EnvVar{
				Name: "QUEUE_CHECKPOINT_WRITES",
				Value: "1",
			},
			corev1.EnvVar{
				Name: "QUEUE_DRAIN",
				Value: "true",
			},
			corev1.EnvVar{
				Name: "QUEUE_MAX_BYTES",
				Value: "1gb",
			},
			corev1.EnvVar{
				Name: "QUEUE_TYPE",
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
