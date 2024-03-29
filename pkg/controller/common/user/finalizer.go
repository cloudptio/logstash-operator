// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package user

import (
	"strings"

	"github.com/cloudptio/logstash-operator/pkg/controller/common/finalizer"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UserFinalizer ensures that any external user created for an associated object is removed.
func UserFinalizer(c k8s.Client, kind string, opts ...client.ListOption) finalizer.Finalizer {
	return finalizer.Finalizer{
		Name: "finalizer.association." + strings.ToLower(kind) + ".k8s.elastic.co/external-user",
		Execute: func() error {
			var secrets corev1.SecretList
			if err := c.List(&secrets, opts...); err != nil {
				return err
			}
			for _, s := range secrets.Items {
				if err := c.Delete(&s); err != nil && !apierrors.IsNotFound(err) {
					return err
				}
			}
			return nil
		},
	}
}
