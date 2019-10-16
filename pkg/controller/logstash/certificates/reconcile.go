// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package certificates

import (
	"time"

	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates/http"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/driver"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/reconciler"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/name"
	coverv1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Reconcile(
	d driver.Interface,
	ls v1beta1.Logstash,
	services []coverv1.Service,
	rotation certificates.RotationParams,
) *reconciler.Results {
	selfSignedCert := ls.Spec.HTTP.TLS.SelfSignedCertificate
	if selfSignedCert != nil && selfSignedCert.Disabled {
		return nil
	}
	results := reconciler.Results{}

	labels := label.NewLabels(ls.Name)

	// reconcile CA certs first
	httpCa, err := certificates.ReconcileCAForOwner(
		d.K8sClient(),
		d.Scheme(),
		name.LSNamer,
		&ls,
		labels,
		certificates.HTTPCAType,
		rotation,
	)
	if err != nil {
		return results.WithError(err)
	}

	// handle CA expiry via requeue
	results.WithResult(reconcile.Result{
		RequeueAfter: certificates.ShouldRotateIn(time.Now(), httpCa.Cert.NotAfter, rotation.RotateBefore),
	})

	// discover and maybe reconcile for the http certificates to use
	httpCertificates, err := http.ReconcileHTTPCertificates(
		d,
		&ls,
		name.LSNamer,
		httpCa,
		ls.Spec.HTTP.TLS,
		labels,
		services,
		rotation, // todo correct rotation
	)
	if err != nil {
		return results.WithError(err)
	}
	// reconcile http public cert secret
	results.WithError(http.ReconcileHTTPCertsPublicSecret(d.K8sClient(), d.Scheme(), &ls, name.LSNamer, httpCertificates))
	return &results
}
