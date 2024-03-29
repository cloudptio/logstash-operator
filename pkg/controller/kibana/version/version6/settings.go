// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version6

import (
	kbtype "github.com/cloudptio/logstash-operator/pkg/apis/kibana/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/kibana/config"
)

// SettingsFactory returns Kibana settings for a 6.x Kibana.
func SettingsFactory(kb kbtype.Kibana) map[string]interface{} {
	return map[string]interface{}{
		config.ElasticsearchURL: kb.AssociationConf().GetURL(),
	}
}
