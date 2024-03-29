// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package driver

import (
	"github.com/cloudptio/logstash-operator/pkg/controller/common/watches"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// Interface describes attributes typically found on reconciler or 'driver' implementations.
type Interface interface {
	K8sClient() k8s.Client
	Scheme() *runtime.Scheme
	DynamicWatches() watches.DynamicWatches
	Recorder() record.EventRecorder
}

// TestDriver is a struct implementing the common driver interface for testing purposes.
type TestDriver struct {
	Client        k8s.Client
	RuntimeScheme *runtime.Scheme
	Watches       watches.DynamicWatches
	FakeRecorder  *record.FakeRecorder
}

func (t TestDriver) K8sClient() k8s.Client {
	return t.Client
}

func (t TestDriver) Scheme() *runtime.Scheme {
	return t.RuntimeScheme
}

func (t TestDriver) DynamicWatches() watches.DynamicWatches {
	return t.Watches
}

func (t TestDriver) Recorder() record.EventRecorder {
	return t.FakeRecorder
}

var _ Interface = &TestDriver{}
