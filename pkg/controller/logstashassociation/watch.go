// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashassociation

import (
	estype "github.com/cloudptio/logstash-operator/pkg/apis/elasticsearch/v1beta1"
	lstype "github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/finalizer"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/watches"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func addWatches(c controller.Controller, r *ReconcileAssociation) error {
	// Watch for changes to Logstash resources
	if err := c.Watch(&source.Kind{Type: &lstype.Logstash{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Dynamically watch related Elasticsearch resources (not all ES resources)
	if err := c.Watch(&source.Kind{Type: &estype.Elasticsearch{}}, r.watches.ElasticsearchClusters); err != nil {
		return err
	}

	// Dynamically watch Elasticsearch public CA secrets for referenced ES clusters
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, r.watches.Secrets); err != nil {
		return err
	}

	// Watch Secrets owned by a Logstash resource
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &lstype.Logstash{},
		IsController: true,
	}); err != nil {
		return err
	}

	return nil
}

// elasticsearchWatchName returns the name of the watch setup on an Elasticsearch cluster
// for a given Logstash resource.
func elasticsearchWatchName(logstashKey types.NamespacedName) string {
	return logstashKey.Namespace + "-" + logstashKey.Name + "-es-watch"
}

// esCAWatchName returns the name of the watch setup on Elasticsearch CA secret
func esCAWatchName(logstash types.NamespacedName) string {
	return logstash.Namespace + "-" + logstash.Name + "-ca-watch"
}

// watchFinalizer ensure that we remove watches for Elasticsearch clusters that we are no longer interested in
// because not referenced by any Logstash resource.
func watchFinalizer(logstashKey types.NamespacedName, w watches.DynamicWatches) finalizer.Finalizer {
	return finalizer.Finalizer{
		Name: "finalizer.association.logstash.k8s.elastic.co/elasticsearch",
		Execute: func() error {
			w.ElasticsearchClusters.RemoveHandlerForKey(elasticsearchWatchName(logstashKey))
			w.Secrets.RemoveHandlerForKey(esCAWatchName(logstashKey))
			return nil
		},
	}
}
