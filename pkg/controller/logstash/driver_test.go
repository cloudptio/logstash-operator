// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstash

import (
	"testing"

	"github.com/cloudptio/logstash-operator/pkg/apis/common/v1beta1"
	lstype "github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates/http"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/deployment"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/version"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/watches"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/pod"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/volume"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var customResourceLimits = corev1.ResourceRequirements{
	Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("2Gi")},
}

func TestDriverDeploymentParams(t *testing.T) {
	s := scheme.Scheme
	if err := lstype.SchemeBuilder.AddToScheme(s); err != nil {
		assert.Fail(t, "failed to build custom scheme")
	}

	type args struct {
		ls             func() *lstype.Logstash
		initialObjects func() []runtime.Object
	}

	tests := []struct {
		name    string
		args    args
		want    deployment.Params
		wantErr bool
	}{
		{
			name: "without remote objects",
			args: args{
				ls:             logstashFixture,
				initialObjects: func() []runtime.Object { return nil },
			},
			want:    deployment.Params{},
			wantErr: true,
		},
		{
			name: "with required remote objects",
			args: args{
				ls:             logstashFixture,
				initialObjects: defaultInitialObjects,
			},
			want:    expectedDeploymentParams(),
			wantErr: false,
		},
		{
			name: "with TLS disabled",
			args: args{
				ls: func() *lstype.Logstash {
					ls := logstashFixture()
					ls.Spec.HTTP.TLS.SelfSignedCertificate = &v1beta1.SelfSignedCertificate{
						Disabled: true,
					}
					return ls
				},
				initialObjects: defaultInitialObjects,
			},
			want: func() deployment.Params {
				params := expectedDeploymentParams()
				params.PodTemplateSpec.Spec.Volumes = params.PodTemplateSpec.Spec.Volumes[:3]
				params.PodTemplateSpec.Spec.Containers[0].VolumeMounts = params.PodTemplateSpec.Spec.Containers[0].VolumeMounts[:3]
				params.PodTemplateSpec.Spec.Containers[0].ReadinessProbe.Handler.HTTPGet.Scheme = corev1.URISchemeHTTP
				return params
			}(),
			wantErr: false,
		},
		{
			name: "with podTemplate specified",
			args: args{
				ls:             logstashFixtureWithPodTemplate,
				initialObjects: defaultInitialObjects,
			},
			want: func() deployment.Params {
				p := expectedDeploymentParams()
				p.PodTemplateSpec.Labels["mylabel"] = "value"
				for i, c := range p.PodTemplateSpec.Spec.Containers {
					if c.Name == lstype.LogstashContainerName {
						p.PodTemplateSpec.Spec.Containers[i].Resources = customResourceLimits
					}
				}
				return p
			}(),
			wantErr: false,
		},
		{
			name: "Checksum takes secret contents into account",
			args: args{
				ls: logstashFixture,
				initialObjects: func() []runtime.Object {
					return []runtime.Object{
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "es-ca-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								certificates.CertFileName: nil,
							},
						},
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-auth",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"logstash-user": []byte("some-secret"),
							},
						},
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-ls-config",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"logstash.yml": []byte("server.name: test"),
							},
						},
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-ls-http-certs-internal",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"tls.crt": []byte("this is also relevant"),
							},
						},
					}
				},
			},
			want: func() deployment.Params {
				p := expectedDeploymentParams()
				p.PodTemplateSpec.Labels = map[string]string{
					"common.k8s.elastic.co/type":              "logstash",
					"logstash.k8s.elastic.co/name":            "test",
					"logstash.k8s.elastic.co/config-checksum": "c5496152d789682387b90ea9b94efcd82a2c6f572f40c016fb86c0d7",
				}
				return p
			}(),
			wantErr: false,
		},
		{
			name: "6.x is supported",
			args: args{
				ls: func() *lstype.Logstash {
					ls := logstashFixture()
					ls.Spec.Version = "6.5.0"
					return ls
				},
				initialObjects: defaultInitialObjects,
			},
			want: func() deployment.Params {
				p := expectedDeploymentParams()
				return p
			}(),
			wantErr: false,
		},
		{
			name: "6.6 docker container already defaults elasticsearch.hosts",
			args: args{
				ls: func() *lstype.Logstash {
					ls := logstashFixture()
					ls.Spec.Version = "6.6.0"
					return ls
				},
				initialObjects: defaultInitialObjects,
			},
			want:    expectedDeploymentParams(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := tt.args.ls()
			initialObjects := tt.args.initialObjects()

			client := k8s.WrapClient(fake.NewFakeClient(initialObjects...))
			w := watches.NewDynamicWatches()
			err := w.Secrets.InjectScheme(scheme.Scheme)
			assert.NoError(t, err)

			lsVersion, err := version.Parse(ls.Spec.Version)
			assert.NoError(t, err)
			d, err := newDriver(client, s, *lsVersion, w, record.NewFakeRecorder(100))
			assert.NoError(t, err)

			got, err := d.deploymentParams(ls)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.Equal(t, tt.want, got)
		})
	}
}

func expectedDeploymentParams() deployment.Params {
	false := false
	return deployment.Params{
		Name:      "test-ls",
		Namespace: "default",
		Selector:  map[string]string{"common.k8s.elastic.co/type": "logstash", "logstash.k8s.elastic.co/name": "test"},
		Labels:    map[string]string{"common.k8s.elastic.co/type": "logstash", "logstash.k8s.elastic.co/name": "test"},
		Replicas:  1,
		PodTemplateSpec: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"common.k8s.elastic.co/type":              "logstash",
					"logstash.k8s.elastic.co/name":            "test",
					"logstash.k8s.elastic.co/config-checksum": "c530a02188193a560326ce91e34fc62dcbd5722b45534a3f60957663",
				},
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: volume.DataVolumeName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "elasticsearch-certs",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "es-ca-secret",
								Optional:   &false,
							},
						},
					},
					{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "test-ls-config",
								Optional:   &false,
							},
						},
					},
					{
						Name: http.HTTPCertificatesSecretVolumeName,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "test-ls-http-certs-internal",
								Optional:   &false,
							},
						},
					},
				},
				Containers: []corev1.Container{{
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volume.DataVolumeName,
							ReadOnly:  false,
							MountPath: volume.DataVolumeMountPath,
						},
						{
							Name:      "elasticsearch-certs",
							ReadOnly:  true,
							MountPath: "/usr/share/logstash/config/elasticsearch-certs",
						},
						{
							Name:      "config",
							ReadOnly:  true,
							MountPath: "/usr/share/logstash/config",
						},
						{
							Name:      http.HTTPCertificatesSecretVolumeName,
							ReadOnly:  true,
							MountPath: http.HTTPCertificatesSecretVolumeMountPath,
						},
					},
					Image: "my-image",
					Name:  lstype.LogstashContainerName,
					Ports: []corev1.ContainerPort{
						{Name: "http", ContainerPort: int32(5601), Protocol: corev1.ProtocolTCP},
					},
					ReadinessProbe: &corev1.Probe{
						FailureThreshold:    3,
						InitialDelaySeconds: 10,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      5,
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Port:   intstr.FromInt(5601),
								Path:   "/login",
								Scheme: corev1.URISchemeHTTPS,
							},
						},
					},
					Resources: pod.DefaultResources,
				}},
				AutomountServiceAccountToken: &false,
			},
		},
	}
}

func logstashFixture() *lstype.Logstash {
	lsFixture := &lstype.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: lstype.LogstashSpec{
			Version: "7.0.0",
			Image:   "my-image",
			Count:   1,
		},
	}

	lsFixture.SetAssociationConf(&v1beta1.AssociationConf{
		AuthSecretName: "test-auth",
		AuthSecretKey:  "logstash-user",
		CASecretName:   "es-ca-secret",
		URL:            "https://localhost:9200",
	})

	return lsFixture
}

func logstashFixtureWithPodTemplate() *lstype.Logstash {
	lsFixture := logstashFixture()
	lsFixture.Spec.PodTemplate = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"mylabel": "value",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:      lstype.LogstashContainerName,
					Resources: customResourceLimits,
				},
			},
		},
	}

	return lsFixture
}

func defaultInitialObjects() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "es-ca-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				certificates.CertFileName: nil,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-auth",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"logstash-user": nil,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ls-config",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"logstash.yml": []byte("server.name: test"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ls-http-certs-internal",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"tls.crt": nil,
			},
		},
	}
}
