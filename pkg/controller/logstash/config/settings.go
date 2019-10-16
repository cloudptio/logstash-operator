// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"path"

	commonv1beta1 "github.com/cloudptio/logstash-operator/pkg/apis/common/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/association"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates/http"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/settings"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/es"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
)

// Logstash configuration settings file
const SettingsFilename = "kibana.yml"

// CanonicalConfig contains configuration for Logstash ("logstash.yml"),
// as a hierarchical key-value configuration.
type CanonicalConfig struct {
	*settings.CanonicalConfig
}

// NewConfigSettings returns the Logstash configuration settings for the given Logstash resource.
func NewConfigSettings(client k8s.Client, ls v1beta1.Logstash) (CanonicalConfig, error) {
	specConfig := ls.Spec.Config
	if specConfig == nil {
		specConfig = &commonv1beta1.Config{}
	}

	userSettings, err := settings.NewCanonicalConfigFrom(specConfig.Data)
	if err != nil {
		return CanonicalConfig{}, err
	}

	username, password, err := association.ElasticsearchAuthSettings(client, &ls)
	if err != nil {
		return CanonicalConfig{}, err
	}

	cfg := settings.MustCanonicalConfig(baseSettings(ls))

	// merge the configuration with userSettings last so they take precedence
	err = cfg.MergeWith(
		settings.MustCanonicalConfig(logstashTLSSettings(ls)),
		settings.MustCanonicalConfig(elasticsearchTLSSettings(ls)),
		settings.MustCanonicalConfig(
			map[string]interface{}{
				ElasticsearchUsername: username,
				ElasticsearchPassword: password,
			},
		),
		userSettings,
	)
	if err != nil {
		return CanonicalConfig{}, err
	}

	return CanonicalConfig{cfg}, nil
}

func baseSettings(ls v1beta1.Logstash) map[string]interface{} {
	return map[string]interface{}{
		ServerName:         ls.Name,
		ServerHost:         "0",
		ElasticSearchHosts: []string{ls.AssociationConf().GetURL()},
		XpackMonitoringUiContainerElasticsearchEnabled: true,
	}
}

func logstashTLSSettings(ls v1beta1.Logstash) map[string]interface{} {
	if !ls.Spec.HTTP.TLS.Enabled() {
		return nil
	}
	return map[string]interface{}{
		ServerSSLEnabled:     true,
		ServerSSLCertificate: path.Join(http.HTTPCertificatesSecretVolumeMountPath, certificates.CertFileName),
		ServerSSLKey:         path.Join(http.HTTPCertificatesSecretVolumeMountPath, certificates.KeyFileName),
	}
}

func elasticsearchTLSSettings(ls v1beta1.Logstash) map[string]interface{} {
	cfg := map[string]interface{}{
		ElasticsearchSslVerificationMode: "certificate",
	}

	if ls.AssociationConf().GetCACertProvided() {
		esCertsVolumeMountPath := es.CaCertSecretVolume(ls).VolumeMount().MountPath
		cfg[ElasticsearchSslCertificateAuthorities] = path.Join(esCertsVolumeMountPath, certificates.CAFileName)
	}

	return cfg
}
