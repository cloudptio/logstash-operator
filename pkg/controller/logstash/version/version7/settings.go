// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version7

import (
	lstype "github.com/elastic/cloud-on-k8s/pkg/apis/logstash/v1beta1"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/config"
)

// SettingsFactory returns Logstash settings for a 7.x Logstash.
func SettingsFactory(ls lstype.Logstash) map[string]interface{} {
	return map[string]interface{}{
		config.ElasticsearchHosts: ls.AssociationConf().GetURL(),
	}
}
