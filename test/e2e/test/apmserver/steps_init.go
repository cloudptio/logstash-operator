// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package apmserver

import (
	"testing"

	apmtype "github.com/cloudptio/logstash-operator/pkg/apis/apm/v1beta1"
	"github.com/cloudptio/logstash-operator/test/e2e/test"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (b Builder) InitTestSteps(k *test.K8sClient) test.StepList {
	return []test.Step{
		{
			Name: "K8S should be accessible",
			Test: func(t *testing.T) {
				pods := corev1.PodList{}
				err := k.Client.List(&pods)
				require.NoError(t, err)
			},
		},

		{
			Name: "APM Server CRDs should exist",
			Test: func(t *testing.T) {
				err := k.Client.List(&apmtype.ApmServerList{})
				require.NoError(t, err)
			},
		},

		{
			Name: "Remove the resources if they already exist",
			Test: func(t *testing.T) {
				for _, obj := range b.RuntimeObjects() {
					err := k.Client.Delete(obj)
					if err != nil {
						// might not exist, which is ok
						require.True(t, apierrors.IsNotFound(err))
					}
				}
				// wait for ES pods to disappear
				test.Eventually(func() error {
					return k.CheckPodCount(0, test.ApmServerPodListOptions(b.ApmServer.Namespace, b.ApmServer.Name)...)
				})(t)
			},
		},
	}
}
