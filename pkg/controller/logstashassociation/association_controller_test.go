// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashassociation

import (
	"testing"

	commonv1beta1 "github.com/cloudptio/logstash-operator/pkg/apis/common/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/apis/elasticsearch/v1beta1"
	estype "github.com/cloudptio/logstash-operator/pkg/apis/elasticsearch/v1beta1"
	kbtype "github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/association"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/user"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	userName       = "default-logstash-foo-logstash-user"
	userSecretName = "logstash-foo-logstash-user" // nolint
)

var esFixture = estype.Elasticsearch{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "es-foo",
		Namespace: "default",
		UID:       "f8d564d9-885e-11e9-896d-08002703f062",
	},
}

var esRefFixture = metav1.OwnerReference{
	APIVersion:         "elasticsearch.k8s.elastic.co/v1beta1",
	Kind:               "Elasticsearch",
	Name:               "es-foo",
	UID:                "f8d564d9-885e-11e9-896d-08002703f062",
	Controller:         &t,
	BlockOwnerDeletion: &t,
}

func setupScheme(t *testing.T) *runtime.Scheme {
	sc := scheme.Scheme
	if err := kbtype.SchemeBuilder.AddToScheme(sc); err != nil {
		assert.Fail(t, "failed to add Logstash types")
	}
	if err := estype.SchemeBuilder.AddToScheme(sc); err != nil {
		assert.Fail(t, "failed to add Es types")
	}
	return sc
}

var logstashFixtureUID types.UID = "82257b19-8862-11e9-896d-08002703f062"

var logstashFixtureObjectMeta = metav1.ObjectMeta{
	Name:      "logstash-foo",
	Namespace: "default",
	UID:       logstashFixtureUID,
}

var logstashFixture = kbtype.Logstash{
	ObjectMeta: logstashFixtureObjectMeta,
	Spec: kbtype.LogstashSpec{
		ElasticsearchRef: commonv1beta1.ObjectSelector{
			Name:      esFixture.Name,
			Namespace: esFixture.Namespace,
		},
	},
}

var t = true
var ownerRefFixture = metav1.OwnerReference{
	APIVersion:         "logstash.k8s.elastic.co/v1beta1",
	Kind:               "Logstash",
	Name:               "foo",
	UID:                logstashFixtureUID,
	Controller:         &t,
	BlockOwnerDeletion: &t,
}

func Test_deleteOrphanedResources(t *testing.T) {
	s := setupScheme(t)
	tests := []struct {
		name           string
		logstash         kbtype.Logstash
		es             v1beta1.Elasticsearch
		initialObjects []runtime.Object
		postCondition  func(c k8s.Client)
		wantErr        bool
	}{
		{
			name: "Do not delete if there's no namespace in the ref",
			logstash: kbtype.Logstash{
				ObjectMeta: logstashFixtureObjectMeta,
				Spec: kbtype.LogstashSpec{
					ElasticsearchRef: commonv1beta1.ObjectSelector{ // ElasticsearchRef without a namespace
						Name: esFixture.Name,
						//Namespace: esFixture.Namespace, No namespace on purpose
					},
				},
			},
			initialObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userSecretName,
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      association.ElasticsearchCACertSecretName(&logstashFixture, ElasticsearchCASecretSuffix),
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							esRefFixture,
						},
						Labels: map[string]string{
							AssociationLabelName: logstashFixture.Name,
							common.TypeLabelName: user.UserType,
						},
					},
				},
			},
			postCondition: func(c k8s.Client) {
				assertExpectObjectsExist(t, c) // all objects must be exist
			},
			wantErr: false,
		},
		{
			name: "ES namespace has changed ",
			logstash: kbtype.Logstash{
				ObjectMeta: logstashFixtureObjectMeta,
				Spec: kbtype.LogstashSpec{
					ElasticsearchRef: commonv1beta1.ObjectSelector{
						Name:      esFixture.Name,
						Namespace: "ns2", // Logstash does not reference the default namespace anymore
					},
				},
			},
			initialObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userSecretName,
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      association.ElasticsearchCACertSecretName(&logstashFixture, ElasticsearchCASecretSuffix),
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: "default", // but we still have a user secret in default
						OwnerReferences: []metav1.OwnerReference{
							esRefFixture,
						},
						Labels: map[string]string{
							AssociationLabelName:      logstashFixture.Name,
							AssociationLabelNamespace: logstashFixture.Namespace,
							common.TypeLabelName:      user.UserType,
						},
					},
				},
			},
			postCondition: func(c k8s.Client) {
				// user CR should be in ES namespace
				assert.Error(t, c.Get(types.NamespacedName{
					Namespace: esFixture.Namespace,
					Name:      userName,
				}, &corev1.Secret{}),
					"Previous user secret should have been removed")
			},
			wantErr: false,
		},
		{
			name:    "nothing to delete",
			logstash:  kbtype.Logstash{},
			wantErr: false,
		},
		{
			name:   "only valid objects",
			logstash: logstashFixture,
			es:     esFixture,
			initialObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userSecretName,
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      association.ElasticsearchCACertSecretName(&logstashFixture, ElasticsearchCASecretSuffix),
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: logstashFixture.Namespace,
						OwnerReferences: []metav1.OwnerReference{
							esRefFixture,
						},
					},
				},
			},
			postCondition: func(c k8s.Client) {
				assertExpectObjectsExist(t, c)
			},
			wantErr: false,
		},
		{
			name: "No more es ref in Logstash, orphan user & CA for previous es ref exist",
			logstash: kbtype.Logstash{
				ObjectMeta: logstashFixtureObjectMeta,
			},
			es: esFixture,
			initialObjects: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userSecretName,
						Namespace: logstashFixture.Namespace,
						Labels: map[string]string{
							AssociationLabelName: logstashFixture.Name,
						},
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: logstashFixture.Namespace,
						Labels: map[string]string{
							AssociationLabelName:      logstashFixture.Name,
							AssociationLabelNamespace: logstashFixture.Namespace,
						},
						OwnerReferences: []metav1.OwnerReference{
							esRefFixture,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      association.ElasticsearchCACertSecretName(&logstashFixture, ElasticsearchCASecretSuffix),
						Namespace: logstashFixture.Namespace,
						Labels: map[string]string{
							AssociationLabelName: logstashFixture.Name,
						},
						OwnerReferences: []metav1.OwnerReference{
							ownerRefFixture,
						},
					},
				},
			},
			postCondition: func(c k8s.Client) {
				// This works even without labels because mock client currently ignores labels
				assert.Error(t, c.Get(types.NamespacedName{
					Namespace: logstashFixture.Namespace,
					Name:      userName,
				}, &corev1.Secret{}))
				assert.Error(t, c.Get(types.NamespacedName{
					Namespace: logstashFixture.Spec.ElasticsearchRef.Namespace,
					Name:      userSecretName,
				}, &corev1.Secret{}))
				assert.Error(t, c.Get(types.NamespacedName{
					Namespace: logstashFixture.Spec.ElasticsearchRef.Namespace,
					Name:      association.ElasticsearchCACertSecretName(&logstashFixture, ElasticsearchCASecretSuffix),
				}, &corev1.Secret{}))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := k8s.WrapClient(fake.NewFakeClientWithScheme(s, tt.initialObjects...))
			if err := deleteOrphanedResources(c, &tt.logstash); (err != nil) != tt.wantErr {
				t.Errorf("deleteOrphanedResources() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.postCondition != nil {
				tt.postCondition(c)
			}
		})
	}
}

func assertExpectObjectsExist(t *testing.T, c k8s.Client) {
	// user CR should be in ES namespace
	assert.NoError(t, c.Get(types.NamespacedName{
		Namespace: esFixture.Namespace,
		Name:      userName,
	}, &corev1.Secret{}))
	// user secret should be in Logstash namespace
	assert.NoError(t, c.Get(types.NamespacedName{
		Namespace: logstashFixture.Namespace,
		Name:      userSecretName,
	}, &corev1.Secret{}))
	// ca secret should be in Logstash namespace
	assert.NoError(t, c.Get(types.NamespacedName{
		Namespace: logstashFixture.Namespace,
		Name:      association.ElasticsearchCACertSecretName(&logstashFixture, ElasticsearchCASecretSuffix),
	}, &corev1.Secret{}))
}
