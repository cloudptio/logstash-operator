// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearch

import (
	"github.com/cloudptio/logstash-operator/pkg/apis/elasticsearch/v1beta1"
	common "github.com/cloudptio/logstash-operator/pkg/controller/common/settings"
	"github.com/cloudptio/logstash-operator/pkg/controller/elasticsearch/settings"
)

func MustNumDataNodes(es v1beta1.Elasticsearch) int {
	var numNodes int
	for _, n := range es.Spec.NodeSets {
		if isDataNode(n) {
			numNodes += int(n.Count)
		}
	}
	return numNodes
}

func isDataNode(node v1beta1.NodeSet) bool {
	if node.Config == nil {
		return v1beta1.DefaultCfg.Node.Data // if not specified use the default
	}
	config, err := common.NewCanonicalConfigFrom(node.Config.Data)
	if err != nil {
		panic(err)
	}
	nodeCfg, err := settings.CanonicalConfig{
		CanonicalConfig: config,
	}.Unpack()
	if err != nil {
		panic(err)
	}
	return nodeCfg.Node.Data
}
