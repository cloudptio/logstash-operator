// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstash

import (
	corev1 "k8s.io/api/core/v1"

	logstashv1beta1 "github.com/elastic/cloud-on-k8s/pkg/apis/logstash/v1beta1"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/defaults"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/label"
	lsname "github.com/elastic/cloud-on-k8s/pkg/controller/logstash/name"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/pod"
)

func NewService(ls logstashv1beta1.Logstash) *corev1.Service {
	svc := corev1.Service{
		ObjectMeta: ls.Spec.HTTP.Service.ObjectMeta,
		Spec:       ls.Spec.HTTP.Service.Spec,
	}

	svc.ObjectMeta.Namespace = ls.Namespace
	svc.ObjectMeta.Name = lsname.HTTPService(ls.Name)

	labels := label.NewLabels(ls.Name)
	ports := []corev1.ServicePort{
		{
			Protocol: corev1.ProtocolTCP,
			Port:     pod.BeatsHTTPPort,
		},
	}

	return defaults.SetServiceDefaults(&svc, labels, labels, ports)
}
