// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package es

import (
	"testing"

	"github.com/cloudptio/logstash-operator/test/e2e/test"
	"github.com/cloudptio/logstash-operator/test/e2e/test/elasticsearch"
)

// TestHTTPWithoutTLS tests an Elasticsearch cluster with TLS disabled for the HTTP layer.
func TestHTTPWithoutTLS(t *testing.T) {
	b := elasticsearch.NewBuilder("test-es-http").
		WithESMasterDataNodes(1, elasticsearch.DefaultResources).
		WithTLSDisabled(true)

	test.Sequence(nil, test.EmptySteps, b).
		RunSequential(t)
}
