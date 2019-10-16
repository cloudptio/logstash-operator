// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"github.com/cloudptio/logstash-operator/cmd/manager"
	"github.com/cloudptio/logstash-operator/pkg/dev"
	"github.com/cloudptio/logstash-operator/pkg/utils/log"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {
	var rootCmd = &cobra.Command{Use: "elastic-operator"}
	rootCmd.AddCommand(manager.Cmd)
	// development mode is only available as a command line flag to avoid accidentally enabling it
	rootCmd.PersistentFlags().BoolVar(&dev.Enabled, "development", false, "turns on development mode")
	log.BindFlags(rootCmd.PersistentFlags())

	cobra.OnInitialize(func() {
		log.InitLogger()
	})

	if err := rootCmd.Execute(); err != nil {
		logf.Log.WithName("main").Error(err, "Unexpected error while executing command")
	}
}
