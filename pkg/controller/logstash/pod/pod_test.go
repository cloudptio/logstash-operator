// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pod

import (
	"testing"

	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/keystore"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_imageWithVersion(t *testing.T) {
	type args struct {
		image   string
		version string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{image: "someimage", version: "6.4.2"},
			want: "someimage:6.4.2",
		},
		{
			args: args{image: "differentimage", version: "6.4.1"},
			want: "differentimage:6.4.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := imageWithVersion(tt.args.image, tt.args.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewPodTemplateSpec(t *testing.T) {
	tests := []struct {
		name       string
		ls         v1beta1.Logstash
		keystore   *keystore.Resources
		assertions func(pod corev1.PodTemplateSpec)
	}{
		{
			name: "defaults",
			ls: v1beta1.Logstash{
				Spec: v1beta1.LogstashSpec{
					Version: "7.1.0",
				},
			},
			keystore: nil,
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Equal(t, false, *pod.Spec.AutomountServiceAccountToken)
				assert.Len(t, pod.Spec.Containers, 1)
				assert.Len(t, pod.Spec.InitContainers, 0)
				assert.Len(t, pod.Spec.Volumes, 1)
				logstashContainer := GetLogstashContainer(pod.Spec)
				require.NotNil(t, logstashContainer)
				assert.Equal(t, 1, len(logstashContainer.VolumeMounts))
				assert.Equal(t, imageWithVersion(defaultImageRepositoryAndName, "7.1.0"), logstashContainer.Image)
				assert.NotNil(t, logstashContainer.ReadinessProbe)
				assert.NotEmpty(t, logstashContainer.Ports)
			},
		},
		{
			name: "with additional volumes and init containers for the Keystore",
			ls: v1beta1.Logstash{
				Spec: v1beta1.LogstashSpec{
					Version: "7.1.0",
				},
			},
			keystore: &keystore.Resources{
				InitContainer: corev1.Container{Name: "init"},
				Volume:        corev1.Volume{Name: "vol"},
			},
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Len(t, pod.Spec.InitContainers, 1)
				assert.Len(t, pod.Spec.Volumes, 2)
			},
		},
		{
			name: "with custom image",
			ls: v1beta1.Logstash{Spec: v1beta1.LogstashSpec{
				Image:   "my-custom-image:1.0.0",
				Version: "7.1.0",
			}},
			keystore: nil,
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Equal(t, "my-custom-image:1.0.0", GetLogstashContainer(pod.Spec).Image)
			},
		},
		{
			name: "with default resources",
			ls: v1beta1.Logstash{Spec: v1beta1.LogstashSpec{
				Version: "7.1.0",
			}},
			keystore: nil,
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Equal(t, DefaultResources, GetLogstashContainer(pod.Spec).Resources)
			},
		},
		{
			name: "with user-provided resources",
			ls: v1beta1.Logstash{Spec: v1beta1.LogstashSpec{
				Version: "7.1.0",
				PodTemplate: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: v1beta1.LogstashContainerName,
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceMemory: resource.MustParse("3Gi"),
									},
								},
							},
						},
					},
				},
			}},
			keystore: nil,
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Equal(t, corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("3Gi"),
					},
				}, GetLogstashContainer(pod.Spec).Resources)
			},
		},
		{
			name: "with user-provided init containers",
			ls: v1beta1.Logstash{Spec: v1beta1.LogstashSpec{
				PodTemplate: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "user-init-container",
							},
						},
					},
				},
			}},
			keystore: nil,
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Len(t, pod.Spec.InitContainers, 1)
			},
		},
		{
			name:     "with user-provided labels",
			keystore: nil,
			ls: v1beta1.Logstash{
				ObjectMeta: metav1.ObjectMeta{
					Name: "logstash-name",
				},
				Spec: v1beta1.LogstashSpec{
					PodTemplate: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"label1":                    "value1",
								"label2":                    "value2",
								label.LogstashNameLabelName: "overridden-logstash-name",
							},
						},
					},
				}},
			assertions: func(pod corev1.PodTemplateSpec) {
				labels := label.NewLabels("logstash-name")
				labels["label1"] = "value1"
				labels["label2"] = "value2"
				labels[label.LogstashNameLabelName] = "overridden-logstash-name"
				assert.Equal(t, labels, pod.Labels)
			},
		},
		{
			name: "with user-provided environment",
			ls: v1beta1.Logstash{Spec: v1beta1.LogstashSpec{
				PodTemplate: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: v1beta1.LogstashContainerName,
								Env: []corev1.EnvVar{
									{
										Name:  "user-env",
										Value: "user-env-value",
									},
								},
							},
						},
					},
				},
			}},
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Len(t, GetLogstashContainer(pod.Spec).Env, 1)
			},
		},
		{
			name: "with user-provided volumes and volume mounts",
			ls: v1beta1.Logstash{Spec: v1beta1.LogstashSpec{
				PodTemplate: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: v1beta1.LogstashContainerName,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name: "user-volume-mount",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "user-volume",
							},
						},
					},
				},
			}},
			assertions: func(pod corev1.PodTemplateSpec) {
				assert.Len(t, pod.Spec.Volumes, 2)
				assert.Len(t, GetLogstashContainer(pod.Spec).VolumeMounts, 2)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPodTemplateSpec(tt.ls, tt.keystore)
			tt.assertions(got)
		})
	}
}
