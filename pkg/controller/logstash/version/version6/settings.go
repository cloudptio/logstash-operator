// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version6

import (
	lstype "github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/config"
)

// SettingsFactory returns Logstash settings for a 6.x Logstash.
func SettingsFactory(ls lstype.Logstash) map[string]interface{} {
	return map[string]interface{}{
		config.ElasticsearchURL: ls.AssociationConf().GetURL(),
	}
}
