// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package es

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/volume"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
)

var eSCertsVolumeMountPath = "/usr/share/kibana/config/elasticsearch-certs"

// CaCertSecretVolume returns a SecretVolume to hold the Elasticsearch CA certs for the given Logstash resource.
func CaCertSecretVolume(ls v1beta1.Logstash) volume.SecretVolume {
	// TODO: this is a little ugly as it reaches into the ES controller bits
	return volume.NewSecretVolumeWithMountPath(
		ls.AssociationConf().GetCASecretName(),
		"elasticsearch-certs",
		eSCertsVolumeMountPath,
	)
}

// GetAuthSecret returns the Elasticsearch auth secret for the given Logstash resource.
func GetAuthSecret(client k8s.Client, ls v1beta1.Logstash) (*corev1.Secret, error) {
	esAuthSecret := types.NamespacedName{
		Name:      ls.AssociationConf().GetAuthSecretName(),
		Namespace: ls.Namespace,
	}
	var secret corev1.Secret
	err := client.Get(esAuthSecret, &secret)
	if err != nil {
		return nil, err
	}
	return &secret, nil
}
