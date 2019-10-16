// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"testing"

	"github.com/cloudptio/logstash-operator/pkg/apis/kibana/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	"github.com/cloudptio/logstash-operator/test/e2e/test"
	"github.com/stretchr/testify/require"
)

func (b Builder) MutationTestSteps(k *test.K8sClient) test.StepList {
	return b.UpgradeTestSteps(k).
		WithSteps(b.CheckK8sTestSteps(k)).
		WithSteps(b.CheckStackTestSteps(k))
}

func (b Builder) MutationReversalTestContext() test.ReversalTestContext {
	panic("not implemented")
}

func (b Builder) UpgradeTestSteps(k *test.K8sClient) test.StepList {
	return test.StepList{
		{
			Name: "Applying the Kibana mutation should succeed",
			Test: func(t *testing.T) {
				var kb v1beta1.Kibana
				require.NoError(t, k.Client.Get(k8s.ExtractNamespacedName(&b.Kibana), &kb))
				kb.Spec = b.Kibana.Spec
				require.NoError(t, k.Client.Update(&kb))
			},
		}}
}
