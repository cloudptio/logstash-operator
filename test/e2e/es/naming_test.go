// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package es

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudptio/logstash-operator/test/e2e/test"
	"github.com/cloudptio/logstash-operator/test/e2e/test/apmserver"
	"github.com/cloudptio/logstash-operator/test/e2e/test/elasticsearch"
	"github.com/cloudptio/logstash-operator/test/e2e/test/kibana"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/validation"

	estype "github.com/cloudptio/logstash-operator/pkg/apis/elasticsearch/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/name"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
)

func TestNameValidation(t *testing.T) {
	t.Run("longestPossibleName", testLongestPossibleName)
	t.Run("rejectionOfLongName", testRejectionOfLongName)
}

func testLongestPossibleName(t *testing.T) {
	maxESNameLen := name.MaxResourceNameLength
	randSuffix := rand.String(4)

	esNamePrefix := strings.Join([]string{"es-naming", randSuffix}, "-")
	esName := strings.Join([]string{esNamePrefix, strings.Repeat("x", maxESNameLen-len(esNamePrefix)-1)}, "-")

	// StatefulSet name would look like <esName>-es-<nodeSpecName>
	// Pods created by the StatefulSet will have the ordinal appended to the name and a controller revision hash
	// label created by appending a revision hash to the pod name.
	revisionHash := fnv.New32()
	_, _ = revisionHash.Write([]byte("some random data"))
	fullRevisionHash := rand.SafeEncodeString(strconv.FormatInt(int64(revisionHash.Sum32()), 10))

	maxNodeSpecNameLen := validation.LabelValueMaxLength - len(esName) - len("-es-") - len("-0") - len(fmt.Sprintf("-%s", fullRevisionHash))
	nodeSpecName := strings.Repeat("y", maxNodeSpecNameLen)
	esBuilder := elasticsearch.NewBuilderWithoutSuffix(esName).
		WithESMasterDataNodes(3, elasticsearch.DefaultResources).
		WithNamespace(test.Ctx().ManagedNamespace(0)).
		WithVersion(test.Ctx().ElasticStackVersion).
		WithNodeSet(estype.NodeSet{
			Name:  nodeSpecName,
			Count: 1,
		}).
		WithRestrictedSecurityContext()

	kbNamePrefix := strings.Join([]string{esNamePrefix, "kb"}, "-")
	kbName := strings.Join([]string{kbNamePrefix, strings.Repeat("x", name.MaxResourceNameLength-len(kbNamePrefix)-1)}, "-")
	kbBuilder := kibana.NewBuilderWithoutSuffix(kbName).
		WithNamespace(test.Ctx().ManagedNamespace(0)).
		WithElasticsearchRef(esBuilder.Ref()).
		WithVersion(test.Ctx().ElasticStackVersion).
		WithRestrictedSecurityContext()

	apmNamePrefix := strings.Join([]string{esNamePrefix, "apm"}, "-")
	apmName := strings.Join([]string{apmNamePrefix, strings.Repeat("x", name.MaxResourceNameLength-len(apmNamePrefix)-1)}, "-")
	apmBuilder := apmserver.NewBuilderWithoutSuffix(apmName).
		WithNamespace(test.Ctx().ManagedNamespace(0)).
		WithElasticsearchRef(esBuilder.Ref()).
		WithVersion(test.Ctx().ElasticStackVersion).
		WithConfig(map[string]interface{}{
			"apm-server.ilm.enabled": false,
		}).
		WithRestrictedSecurityContext()

	test.Sequence(nil, test.EmptySteps, esBuilder, kbBuilder, apmBuilder).RunSequential(t)
}

func testRejectionOfLongName(t *testing.T) {
	k := test.NewK8sClientOrFatal()

	randSuffix := rand.String(4)
	esName := strings.Join([]string{"es-name-length", randSuffix, strings.Repeat("x", name.MaxResourceNameLength)}, "-")
	esBuilder := elasticsearch.NewBuilderWithoutSuffix(esName).
		WithESMasterDataNodes(1, elasticsearch.DefaultResources).
		WithNamespace(test.Ctx().ManagedNamespace(0)).
		WithVersion(test.Ctx().ElasticStackVersion).
		WithNodeSet(estype.NodeSet{
			Name:  "default",
			Count: 1,
		}).
		WithRestrictedSecurityContext()

	objectCreated := false

	testSteps := test.StepList{
		test.Step{
			Name: "Creating an Elasticsearch object should fail validation",
			Test: func(t *testing.T) {
				for _, obj := range esBuilder.RuntimeObjects() {
					err := k.Client.Create(obj)
					if err != nil {
						// validating webhook is active and rejected the request
						require.Contains(t, err.Error(), `admission webhook "validation.elasticsearch.elastic.co" denied the request`)
						return
					}

					// if the validating webhook is not active, operator's own validation check should set the object phase to "Invalid"
					objectCreated = true
					test.Eventually(func() error {
						var createdES estype.Elasticsearch
						if err := k.Client.Get(k8s.ExtractNamespacedName(&esBuilder.Elasticsearch), &createdES); err != nil {
							return err
						}

						if createdES.Status.Phase != estype.ElasticsearchResourceInvalid {
							return fmt.Errorf("expected phase=[%s], actual phase=[%s]", estype.ElasticsearchResourceInvalid, createdES.Status.Phase)
						}
						return nil
					})
				}
			},
		},
		test.Step{
			Name: "Deleting an invalid Elasticsearch object should succeed",
			Test: func(t *testing.T) {
				// if the validating webhook rejected the request, we have nothing to delete
				if !objectCreated {
					return
				}

				for _, obj := range esBuilder.RuntimeObjects() {
					err := k.Client.Delete(obj)
					require.NoError(t, err)
				}

				test.Eventually(func() error {
					var createdES estype.Elasticsearch
					err := k.Client.Get(k8s.ExtractNamespacedName(&esBuilder.Elasticsearch), &createdES)
					if apierrors.IsNotFound(err) {
						return nil
					}

					return errors.Wrapf(err, "object should not exist")
				})
			},
		},
	}

	testSteps.RunSequential(t)
}
