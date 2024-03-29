// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashassociation

import (
	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/user"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// AssociationLabelName marks resources created by this controller for easier retrieval.
	AssociationLabelName = "logstashassociation.k8s.elastic.co/name"
	// AssociationLabelNamespace marks resources created by this controller for easier retrieval.
	AssociationLabelNamespace = "logstashassociation.k8s.elastic.co/namespace"
)

// NewResourceSelector selects resources labeled as related to the named association.
func NewResourceSelector(name string) client.MatchingLabels {
	return client.MatchingLabels(map[string]string{
		AssociationLabelName: name,
	})
}

func hasBeenCreatedBy(object metav1.Object, logstash *v1beta1.Logstash) bool {
	labels := object.GetLabels()
	if name, ok := labels[AssociationLabelName]; !ok || name != logstash.Name {
		return false
	}
	if ns, ok := labels[AssociationLabelNamespace]; !ok || ns != logstash.Namespace {
		return false
	}
	return true
}

func NewUserLabelSelector(
	namespacedName types.NamespacedName,
) client.MatchingLabels {
	return client.MatchingLabels(
		map[string]string{
			AssociationLabelName:      namespacedName.Name,
			AssociationLabelNamespace: namespacedName.Namespace,
			common.TypeLabelName:      user.UserType,
		})
}
