// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package association

import (
	"reflect"

	commonv1beta1 "github.com/cloudptio/logstash-operator/pkg/apis/common/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates/http"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/reconciler"
	esname "github.com/cloudptio/logstash-operator/pkg/controller/elasticsearch/name"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// CASecret is a container to hold information about the Elasticsearch CA secret.
type CASecret struct {
	Name           string
	CACertProvided bool
}

// ElasticsearchCACertSecretName returns the name of the secret holding the certificate chain used
// by the associated resource to establish and validate a secured HTTP connection to Elasticsearch.
func ElasticsearchCACertSecretName(associated commonv1beta1.Associated, suffix string) string {
	return associated.GetName() + "-" + suffix
}

// ReconcileCASecret keeps in sync a copy of the Elasticsearch CA.
// It is the responsibility of the controller to set a watch on the ES CA.
func ReconcileCASecret(
	client k8s.Client,
	scheme *runtime.Scheme,
	associated commonv1beta1.Associated,
	es types.NamespacedName,
	labels map[string]string,
	suffix string,
) (CASecret, error) {
	publicESHTTPCertificatesNSN := http.PublicCertsSecretRef(esname.ESNamer, es)

	// retrieve the HTTP certificates from ES namespace
	var publicESHTTPCertificatesSecret corev1.Secret
	if err := client.Get(publicESHTTPCertificatesNSN, &publicESHTTPCertificatesSecret); err != nil {
		if errors.IsNotFound(err) {
			return CASecret{}, nil // probably not created yet, we'll be notified to reconcile later
		}
		return CASecret{}, err
	}

	// Certificate data should be copied over a secret in the associated namespace
	expectedSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: associated.GetNamespace(),
			Name:      ElasticsearchCACertSecretName(associated, suffix),
			Labels:    labels,
		},
		Data: publicESHTTPCertificatesSecret.Data,
	}
	var reconciledSecret corev1.Secret
	if err := reconciler.ReconcileResource(reconciler.Params{
		Client:     client,
		Scheme:     scheme,
		Owner:      associated,
		Expected:   &expectedSecret,
		Reconciled: &reconciledSecret,
		NeedsUpdate: func() bool {
			return !reflect.DeepEqual(expectedSecret.Data, reconciledSecret.Data)
		},
		UpdateReconciled: func() {
			reconciledSecret.Data = expectedSecret.Data
		},
	}); err != nil {
		return CASecret{}, err
	}

	_, caCertProvided := expectedSecret.Data[certificates.CAFileName]
	return CASecret{Name: expectedSecret.Name, CACertProvided: caCertProvided}, nil
}
