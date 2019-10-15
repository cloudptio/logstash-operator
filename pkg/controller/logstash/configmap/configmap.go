// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configmap

import (
	"bytes"
	"html/template"

	"github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/association"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/name"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// NewConfigMapWithData constructs a new config map with the given data
func NewConfigMapWithData(ls types.NamespacedName, data map[string]string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ls.Name,
			Namespace: ls.Namespace,
			Labels:    label.NewLabels(ls.Name),
		},
		Data: data,
	}
}

var inputConfTemplateStr = `input {
  # udp {
  #   port => 1514
  #   type => syslog
  # }
  # tcp {
  #   port => 1514
  #   type => syslog
  # }
  beats {
    port => 5044
  }
  # http {
  #   port => 8080
  # }
  # kafka {
  #   ## ref: https://www.elastic.co/guide/en/logstash/current/plugins-inputs-kafka.html
  #   bootstrap_servers => "kafka-input:9092"
  #   codec => json { charset => "UTF-8" }
  #   consumer_threads => 1
  #   topics => ["source"]
  #   type => "example"
  # }
}`
var outputConfTemplateStr = `output {
	# stdout { codec => rubydebug }
	elasticsearch {
		hosts => ["{{ .ElasticsearchHost }}"]
		user => "{{ .Username }}"
		password => "{{ .Password }}"
		manage_template => false
		index => "%{[@metadata][beat]}-%{+YYYY.MM.dd}"
		ssl => true
		cacert => "/usr/share/kibana/config/elasticsearch-certs/tls.crt"
	}
}`

var inputConfTemplate *template.Template
var outputConfTemplate *template.Template

func init() {
	var err error
	inputConfTemplate, err = template.New("input").Parse(inputConfTemplateStr)
	if err != nil {
		panic(err)
	}
	outputConfTemplate, err = template.New("output").Parse(outputConfTemplateStr)
	if err != nil {
		panic(err)
	}
}

type confStruct struct {
	ElasticsearchHost string
	Username          string
	Password          string
}

// ReconcileScriptsConfigMap reconciles a configmap containing pipeline input
// and output files.
func ReconcilePipelineConfigMap(c k8s.Client, scheme *runtime.Scheme, ls v1beta1.Logstash) error {
	username, password, err := association.ElasticsearchAuthSettings(c, &ls)
	if err != nil {
		return err
	}
	conf := confStruct{
		ls.AssociationConf().GetURL(),
		username,
		password,
	}
	if ls.Spec.InputConf == "" {
		var buf bytes.Buffer
		if err := inputConfTemplate.Execute(&buf, conf); err != nil {
			return err
		}
		ls.Spec.InputConf = buf.String()
	}
	if ls.Spec.OutputConf == "" {
		var buf bytes.Buffer
		if err := outputConfTemplate.Execute(&buf, conf); err != nil {
			return err
		}
		ls.Spec.OutputConf = buf.String()
	}

	pipelineConfigmap := NewConfigMapWithData(
		types.NamespacedName{Namespace: ls.Namespace, Name: name.PipelineConfigMap(ls.Name)},
		map[string]string{
			"input_main.conf":  ls.Spec.InputConf,
			"output_main.conf": ls.Spec.OutputConf,
		},
	)

	return ReconcileConfigMap(c, scheme, ls, pipelineConfigmap)
}
