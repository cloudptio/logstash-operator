// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstash

import (
	"crypto/sha256"
	"fmt"

	lstype "github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates/http"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/deployment"
	driver2 "github.com/cloudptio/logstash-operator/pkg/controller/common/driver"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/events"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/finalizer"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/keystore"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/operator"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/reconciler"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/version"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/watches"
	lscerts "github.com/cloudptio/logstash-operator/pkg/controller/logstash/certificates"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/configmap"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/es"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	lsname "github.com/cloudptio/logstash-operator/pkg/controller/logstash/name"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/pod"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/version/version6"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/version/version7"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/volume"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// initContainersParameters is used to generate the init container that will load the secure settings into a keystore
var initContainersParameters = keystore.InitContainerParameters{
	KeystoreCreateCommand:         "/usr/share/kibana/bin/logstash-keystore create",
	KeystoreAddCommand:            `/usr/share/kibana/bin/logstash-keystore add "$key" --stdin < "$filename"`,
	SecureSettingsVolumeMountPath: keystore.SecureSettingsVolumeMountPath,
	DataVolumePath:                volume.DataVolumeMountPath,
}

type driver struct {
	client          k8s.Client
	scheme          *runtime.Scheme
	settingsFactory func(ls lstype.Logstash) map[string]interface{}
	dynamicWatches  watches.DynamicWatches
	recorder        record.EventRecorder
}

func (d *driver) DynamicWatches() watches.DynamicWatches {
	return d.dynamicWatches
}

func (d *driver) K8sClient() k8s.Client {
	return d.client
}

func (d *driver) Recorder() record.EventRecorder {
	return d.recorder
}

func (d *driver) Scheme() *runtime.Scheme {
	return d.scheme
}

var _ driver2.Interface = &driver{}

func secretWatchKey(logstash lstype.Logstash) string {
	return fmt.Sprintf("%s-%s-es-auth-secret", logstash.Namespace, logstash.Name)
}

func secretWatchFinalizer(logstash lstype.Logstash, watches watches.DynamicWatches) finalizer.Finalizer {
	return finalizer.Finalizer{
		Name: "finalizer.logstash.k8s.elastic.co/es-auth-secret",
		Execute: func() error {
			watches.Secrets.RemoveHandlerForKey(secretWatchKey(logstash))
			return nil
		},
	}
}

func (d *driver) deploymentParams(ls *lstype.Logstash) (deployment.Params, error) {
	// setup a keystore with secure settings in an init container, if specified by the user
	keystoreResources, err := keystore.NewResources(
		d,
		ls,
		lsname.LSNamer,
		label.NewLabels(ls.Name),
		initContainersParameters,
	)
	if err != nil {
		return deployment.Params{}, err
	}

	logstashPodSpec := pod.NewPodTemplateSpec(*ls, keystoreResources)

	// TODO: Add reference to dynamic ES connection
	//logstashPodSpec.ls.AssociationConf().URL

	// Build a checksum of the configuration, which we can use to cause the Deployment to roll Logstash
	// instances in case of any change in the CA file, secure settings or credentials contents.
	// This is done because Logstash does not support updating those without restarting the process.
	configChecksum := sha256.New224()
	if keystoreResources != nil {
		_, _ = configChecksum.Write([]byte(keystoreResources.Version))
	}

	// we need to deref the secret here (if any) to include it in the checksum otherwise Logstash will not be rolled on contents changes
	if ls.AssociationConf().AuthIsConfigured() {
		esAuthSecret := types.NamespacedName{Name: ls.AssociationConf().GetAuthSecretName(), Namespace: ls.Namespace}
		if err := d.dynamicWatches.Secrets.AddHandler(watches.NamedWatch{
			Name:    secretWatchKey(*ls),
			Watched: []types.NamespacedName{esAuthSecret},
			Watcher: k8s.ExtractNamespacedName(ls),
		}); err != nil {
			return deployment.Params{}, err
		}
		sec := corev1.Secret{}
		if err := d.client.Get(esAuthSecret, &sec); err != nil {
			return deployment.Params{}, err
		}
		_, _ = configChecksum.Write(sec.Data[ls.AssociationConf().GetAuthSecretKey()])
	} else {
		d.dynamicWatches.Secrets.RemoveHandlerForKey(secretWatchKey(*ls))
	}

	if ls.AssociationConf().CAIsConfigured() {
		var esPublicCASecret corev1.Secret
		key := types.NamespacedName{Namespace: ls.Namespace, Name: ls.AssociationConf().GetCASecretName()}
		// watch for changes in the CA secret
		if err := d.dynamicWatches.Secrets.AddHandler(watches.NamedWatch{
			Name:    secretWatchKey(*ls),
			Watched: []types.NamespacedName{key},
			Watcher: k8s.ExtractNamespacedName(ls),
		}); err != nil {
			return deployment.Params{}, err
		}

		if err := d.client.Get(key, &esPublicCASecret); err != nil {
			return deployment.Params{}, err
		}
		if certPem, ok := esPublicCASecret.Data[certificates.CertFileName]; ok {
			_, _ = configChecksum.Write(certPem)
		}

		// TODO: this is a little ugly as it reaches into the ES controller bits
		esCertsVolume := es.CaCertSecretVolume(*ls)

		logstashPodSpec.Spec.Volumes = append(logstashPodSpec.Spec.Volumes,
			esCertsVolume.Volume())

		for i := range logstashPodSpec.Spec.InitContainers {
			logstashPodSpec.Spec.InitContainers[i].VolumeMounts = append(logstashPodSpec.Spec.InitContainers[i].VolumeMounts,
				esCertsVolume.VolumeMount())
		}

		logstashContainer := pod.GetLogstashContainer(logstashPodSpec.Spec)
		logstashContainer.VolumeMounts = append(logstashContainer.VolumeMounts,
			esCertsVolume.VolumeMount())
	}

	if ls.Spec.HTTP.TLS.Enabled() {
		// fetch the secret to calculate the checksum
		var httpCerts corev1.Secret
		err := d.client.Get(types.NamespacedName{
			Namespace: ls.Namespace,
			Name:      certificates.HTTPCertsInternalSecretName(lsname.LSNamer, ls.Name),
		}, &httpCerts)
		if err != nil {
			return deployment.Params{}, err
		}
		if httpCert, ok := httpCerts.Data[certificates.CertFileName]; ok {
			_, _ = configChecksum.Write(httpCert)
		}

		// add volume/mount for http certs to pod spec
		httpCertsVolume := http.HTTPCertSecretVolume(lsname.LSNamer, ls.Name)
		logstashPodSpec.Spec.Volumes = append(logstashPodSpec.Spec.Volumes, httpCertsVolume.Volume())
		logstashContainer := pod.GetLogstashContainer(logstashPodSpec.Spec)
		logstashContainer.VolumeMounts = append(logstashContainer.VolumeMounts, httpCertsVolume.VolumeMount())

	}

	// // get config secret to add its content to the config checksum
	// configSecret := corev1.Secret{}
	// err = d.client.Get(types.NamespacedName{Name: config.SecretName(*ls), Namespace: ls.Namespace}, &configSecret)
	// if err != nil {
	// 	return deployment.Params{}, err
	// }
	// _, _ = configChecksum.Write(configSecret.Data[config.SettingsFilename])

	// add the checksum to a label for the deployment and its pods (the important bit is that the pod template
	// changes, which will trigger a rolling update)
	logstashPodSpec.Labels[configChecksumLabel] = fmt.Sprintf("%x", configChecksum.Sum(nil))

	return deployment.Params{
		Name:            lsname.LSNamer.Suffix(ls.Name),
		Namespace:       ls.Namespace,
		Replicas:        ls.Spec.Count,
		Selector:        label.NewLabels(ls.Name),
		Labels:          label.NewLabels(ls.Name),
		PodTemplateSpec: logstashPodSpec,
	}, nil
}

func (d *driver) Reconcile(
	state *State,
	ls *lstype.Logstash,
	params operator.Parameters,
) *reconciler.Results {
	results := reconciler.Results{}
	if !ls.AssociationConf().IsConfigured() {
		d.recorder.Event(ls, corev1.EventTypeWarning, events.EventAssociationError, "Elasticsearch backend is not configured")
		log.Info("Aborting Logstash deployment reconciliation as no Elasticsearch backend is configured", "namespace", ls.Namespace, "logstash_name", ls.Name)
		return &results
	}

	if err := configmap.ReconcilePipelineConfigMap(d.client, d.scheme, *ls); err != nil {
		return results.WithError(err)
	}

	svc, err := common.ReconcileService(d.client, d.scheme, NewService(*ls), ls)
	if err != nil {
		// TODO: consider updating some status here?
		return results.WithError(err)
	}

	results.WithResults(lscerts.Reconcile(d, *ls, []corev1.Service{*svc}, params.CACertRotation))
	if results.HasError() {
		return &results
	}

	// lsSettings, err := config.NewConfigSettings(d.client, *ls)
	// if err != nil {
	// 	return results.WithError(err)
	// }
	// err = lsSettings.MergeWith(
	// 	settings.MustCanonicalConfig(d.settingsFactory(*ls)),
	// )
	// if err != nil {
	// 	return results.WithError(err)
	// }
	// err = config.ReconcileConfigSecret(d.client, *ls, lsSettings, params.OperatorInfo)
	// if err != nil {
	// 	return results.WithError(err)
	// }

	deploymentParams, err := d.deploymentParams(ls)
	if err != nil {
		return results.WithError(err)
	}
	expectedDp := deployment.New(deploymentParams)
	reconciledDp, err := deployment.Reconcile(d.client, d.scheme, expectedDp, ls)
	if err != nil {
		return results.WithError(err)
	}
	state.UpdateLogstashState(reconciledDp)
	return &results
}

func newDriver(
	client k8s.Client,
	scheme *runtime.Scheme,
	version version.Version,
	watches watches.DynamicWatches,
	recorder record.EventRecorder,
) (*driver, error) {
	d := driver{
		client:         client,
		scheme:         scheme,
		dynamicWatches: watches,
		recorder:       recorder,
	}
	switch version.Major {
	case 6:
		switch {
		case version.Minor >= 6:
			// 6.6 docker container already defaults to v7 settings
			d.settingsFactory = version7.SettingsFactory
		default:
			d.settingsFactory = version6.SettingsFactory
		}
	case 7:
		d.settingsFactory = version7.SettingsFactory
	default:
		return nil, fmt.Errorf("unsupported version: %s", version)
	}
	return &d, nil
}
