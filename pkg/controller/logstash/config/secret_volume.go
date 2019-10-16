// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/volume"
)

// Constants to use for the config files in a Logstash pod.
const (
	VolumeName      = "config"
	VolumeMountPath = "/usr/share/kibana/" + VolumeName
)

// SecretVolume returns a SecretVolume to hold the Logstash config of the given Logstash resource.
func SecretVolume(ls v1beta1.Logstash) volume.SecretVolume {
	return volume.NewSecretVolumeWithMountPath(
		SecretName(ls),
		VolumeName,
		VolumeMountPath,
	)
}

// SecretName is the name of the secret that holds the Logstash config for the given Logstash resource.
func SecretName(ls v1beta1.Logstash) string {
	return ls.Name + "-ls-" + VolumeName
}
