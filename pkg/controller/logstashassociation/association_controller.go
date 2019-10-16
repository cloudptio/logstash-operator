// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashassociation

import (
	"reflect"
	"time"

	commonv1beta1 "github.com/cloudptio/logstash-operator/pkg/apis/common/v1beta1"
	estype "github.com/cloudptio/logstash-operator/pkg/apis/elasticsearch/v1beta1"
	lstype "github.com/cloudptio/logstash-operator/pkg/apis/logstash/v1beta1"
	"github.com/cloudptio/logstash-operator/pkg/controller/common"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/annotation"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/association"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/certificates/http"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/events"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/finalizer"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/operator"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/user"
	"github.com/cloudptio/logstash-operator/pkg/controller/common/watches"
	esname "github.com/cloudptio/logstash-operator/pkg/controller/elasticsearch/name"
	"github.com/cloudptio/logstash-operator/pkg/controller/elasticsearch/services"
	elasticsearchuser "github.com/cloudptio/logstash-operator/pkg/controller/elasticsearch/user"
	"github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	lslabel "github.com/cloudptio/logstash-operator/pkg/controller/logstash/label"
	"github.com/cloudptio/logstash-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Logstash association controller
//
// This controller's only purpose is to complete a Logstash resource
// with connection details to an existing Elasticsearch cluster.
//
// High-level overview:
// - watch Logstash resources
// - if a Logstash resource specifies an Elasticsearch resource reference,
//   resolve details about that ES cluster (url, credentials), and update
//   the Logstash resource with ES connection details
// - create the Logstash user for Elasticsearch
// - copy ES CA public cert secret into Logstash namespace
// - reconcile on any change from watching Logstash, Elasticsearch, Users and secrets
//
// If reference to an Elasticsearch cluster is not set in the Logstash resource,
// this controller does nothing.

const (
	name = "logstash-association-controller"
	// logstashUserSuffix is used to suffix user and associated secret resources.
	logstashUserSuffix = "logstash-user"
	// ElasticsearchCASecretSuffix is used as suffix for CAPublicCertSecretName
	ElasticsearchCASecretSuffix = "ls-es-ca" // nolint
)

var (
	log            = logf.Log.WithName(name)
	defaultRequeue = reconcile.Result{Requeue: true, RequeueAfter: 10 * time.Second}
)

// Add creates a new Association Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, params operator.Parameters) error {
	r := newReconciler(mgr, params)
	c, err := add(mgr, r)
	if err != nil {
		return err
	}
	return addWatches(c, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, params operator.Parameters) *ReconcileAssociation {
	client := k8s.WrapClient(mgr.GetClient())
	return &ReconcileAssociation{
		Client:     client,
		scheme:     mgr.GetScheme(),
		watches:    watches.NewDynamicWatches(),
		recorder:   mgr.GetEventRecorderFor(name),
		Parameters: params,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) (controller.Controller, error) {
	// Create a new controller
	c, err := controller.New(name, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return c, err
	}
	return c, nil
}

var _ reconcile.Reconciler = &ReconcileAssociation{}

// ReconcileAssociation reconciles a Logstash resource for association with Elasticsearch
type ReconcileAssociation struct {
	k8s.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	watches  watches.DynamicWatches
	operator.Parameters
	// iteration is the number of times this controller has run its Reconcile method
	iteration uint64
}

// Reconcile reads that state of the cluster for an Association object and makes changes based on the state read and what is in
// the Association.Spec
func (r *ReconcileAssociation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	defer common.LogReconciliationRun(log, request, &r.iteration)()

	var logstash lstype.Logstash
	if ok, err := association.FetchWithAssociation(r.Client, request, &logstash); !ok {
		return reconcile.Result{}, err
	}

	// register or execute watch finalizers
	h := finalizer.NewHandler(r)
	lsName := k8s.ExtractNamespacedName(&logstash)
	err := h.Handle(
		&logstash,
		watchFinalizer(lsName, r.watches),
		user.UserFinalizer(r.Client, logstash.Kind, NewUserLabelSelector(lsName)),
	)
	if err != nil {
		if apierrors.IsConflict(err) {
			// Conflicts are expected here and should be resolved on next loop
			log.V(1).Info("Conflict while handling finalizer")
			return reconcile.Result{Requeue: true}, nil
		}
		// failed to prepare or run finalizer: retry
		return defaultRequeue, err
	}

	// Logstash is being deleted: short-circuit reconciliation
	if !logstash.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	if common.IsPaused(logstash.ObjectMeta) {
		log.Info("Object is paused. Skipping reconciliation", "namespace", logstash.Namespace, "logstash_name", logstash.Name)
		return common.PauseRequeue, nil
	}

	compatible, err := r.isCompatible(&logstash)
	if err != nil || !compatible {
		return reconcile.Result{}, err
	}

	newStatus, err := r.reconcileInternal(&logstash)
	if err != nil {
		k8s.EmitErrorEvent(r.recorder, err, &logstash, events.EventReconciliationError, "Reconciliation error: %v", err)
	}

	// maybe update status
	if !reflect.DeepEqual(logstash.Status.AssociationStatus, newStatus) {
		oldStatus := logstash.Status.AssociationStatus
		logstash.Status.AssociationStatus = newStatus
		if err := r.Status().Update(&logstash); err != nil {
			if apierrors.IsConflict(err) {
				// Conflicts are expected and will be resolved on next loop
				log.V(1).Info("Conflict while updating status", "namespace", logstash.Namespace, "logstash_name", logstash.Name)
				return reconcile.Result{Requeue: true}, nil
			}

			return defaultRequeue, err
		}
		r.recorder.AnnotatedEventf(&logstash,
			annotation.ForAssociationStatusChange(oldStatus, newStatus),
			corev1.EventTypeNormal,
			events.EventAssociationStatusChange,
			"Association status changed from [%s] to [%s]", oldStatus, newStatus)
	}
	return resultFromStatus(newStatus), err
}

func resultFromStatus(status commonv1beta1.AssociationStatus) reconcile.Result {
	switch status {
	case commonv1beta1.AssociationPending:
		return defaultRequeue // retry
	default:
		return reconcile.Result{} // we are done or there is not much we can do
	}
}

func (r *ReconcileAssociation) isCompatible(logstash *lstype.Logstash) (bool, error) {
	selector := map[string]string{label.LogstashNameLabelName: logstash.Name}
	compat, err := annotation.ReconcileCompatibility(r.Client, logstash, selector, r.OperatorInfo.BuildInfo.Version)
	if err != nil {
		k8s.EmitErrorEvent(r.recorder, err, logstash, events.EventCompatCheckError, "Error during compatibility check: %v", err)
	}

	return compat, err
}

func (r *ReconcileAssociation) reconcileInternal(logstash *lstype.Logstash) (commonv1beta1.AssociationStatus, error) {
	logstashKey := k8s.ExtractNamespacedName(logstash)

	// garbage collect leftover resources that are not required anymore
	if err := deleteOrphanedResources(r, logstash); err != nil {
		log.Error(err, "Error while trying to delete orphaned resources. Continuing.", "namespace", logstash.Namespace, "logstash_name", logstash.Name)
	}

	if logstash.Spec.ElasticsearchRef.Name == "" {
		// stop watching any ES cluster previously referenced for this Logstash resource
		r.watches.ElasticsearchClusters.RemoveHandlerForKey(elasticsearchWatchName(logstashKey))
		// other leftover resources are already garbage-collected
		return commonv1beta1.AssociationUnknown, nil
	}

	// this Logstash instance references an Elasticsearch cluster
	esRef := logstash.Spec.ElasticsearchRef
	if esRef.Namespace == "" {
		// no namespace provided: default to Logstash's namespace
		esRef.Namespace = logstash.Namespace
	}
	esRefKey := esRef.NamespacedName()

	// watch the referenced ES cluster for future reconciliations
	if err := r.watches.ElasticsearchClusters.AddHandler(watches.NamedWatch{
		Name:    elasticsearchWatchName(logstashKey),
		Watched: []types.NamespacedName{esRefKey},
		Watcher: logstashKey,
	}); err != nil {
		return commonv1beta1.AssociationFailed, err
	}

	userSecretKey := association.UserKey(logstash, logstashUserSuffix)
	// watch the user secret in the ES namespace
	if err := r.watches.Secrets.AddHandler(watches.NamedWatch{
		Name:    elasticsearchWatchName(logstashKey),
		Watched: []types.NamespacedName{userSecretKey},
		Watcher: logstashKey,
	}); err != nil {
		return commonv1beta1.AssociationFailed, err
	}

	var es estype.Elasticsearch
	if err := r.Get(esRefKey, &es); err != nil {
		k8s.EmitErrorEvent(r.recorder, err, logstash, events.EventAssociationError, "Failed to find referenced backend %s: %v", esRefKey, err)
		if apierrors.IsNotFound(err) {
			// ES not found. 2 options:
			// - not created yet: that's ok, we'll reconcile on creation event
			// - deleted: existing resources will be garbage collected
			// in any case, since the user explicitly requested a managed association,
			// remove connection details if they are set
			if err := association.RemoveAssociationConf(r.Client, logstash); err != nil && !errors.IsConflict(err) {
				log.Error(err, "Failed to remove Elasticsearch configuration from Logstash object",
					"namespace", logstash.Namespace, "logstash_name", logstash.Name)
				return commonv1beta1.AssociationPending, err
			}

			return commonv1beta1.AssociationPending, nil
		}
		return commonv1beta1.AssociationFailed, err
	}

	if err := association.ReconcileEsUser(
		r.Client,
		r.scheme,
		logstash,
		map[string]string{
			AssociationLabelName:      logstash.Name,
			AssociationLabelNamespace: logstash.Namespace,
		},
		elasticsearchuser.LogstashSystemUserBuiltinRole,
		logstashUserSuffix,
		es); err != nil {
		return commonv1beta1.AssociationPending, err
	}

	caSecret, err := r.reconcileElasticsearchCA(logstash, esRefKey)
	if err != nil {
		return commonv1beta1.AssociationPending, err
	}

	// construct the expected association configuration
	authSecret := association.ClearTextSecretKeySelector(logstash, logstashUserSuffix)
	expectedESAssoc := &commonv1beta1.AssociationConf{
		AuthSecretName: authSecret.Name,
		AuthSecretKey:  authSecret.Key,
		CACertProvided: caSecret.CACertProvided,
		CASecretName:   caSecret.Name,
		URL:            services.ExternalServiceURL(es),
	}

	// update the association configuration if necessary
	if !reflect.DeepEqual(expectedESAssoc, logstash.AssociationConf()) {
		log.Info("Updating Logstash spec with Elasticsearch backend configuration", "namespace", logstash.Namespace, "logstash_name", logstash.Name)
		if err := association.UpdateAssociationConf(r.Client, logstash, expectedESAssoc); err != nil {
			if errors.IsConflict(err) {
				return commonv1beta1.AssociationPending, nil
			}
			log.Error(err, "Failed to update association configuration", "namespace", logstash.Namespace, "logstash_name", logstash.Name)
			return commonv1beta1.AssociationPending, err
		}
		logstash.SetAssociationConf(expectedESAssoc)
	}

	return commonv1beta1.AssociationEstablished, nil
}

func (r *ReconcileAssociation) reconcileElasticsearchCA(logstash *lstype.Logstash, es types.NamespacedName) (association.CASecret, error) {
	logstashKey := k8s.ExtractNamespacedName(logstash)
	// watch ES CA secret to reconcile on any change
	if err := r.watches.Secrets.AddHandler(watches.NamedWatch{
		Name:    esCAWatchName(logstashKey),
		Watched: []types.NamespacedName{http.PublicCertsSecretRef(esname.ESNamer, es)},
		Watcher: logstashKey,
	}); err != nil {
		return association.CASecret{}, err
	}
	// Build the labels applied on the secret
	labels := lslabel.NewLabels(logstash.Name)
	labels[AssociationLabelName] = logstash.Name
	return association.ReconcileCASecret(
		r.Client,
		r.scheme,
		logstash,
		es,
		labels,
		ElasticsearchCASecretSuffix,
	)
}

// deleteOrphanedResources deletes resources created by this association that are left over from previous reconciliation
// attempts. Common use case is an Elasticsearch reference in Logstash spec that was removed.
func deleteOrphanedResources(c k8s.Client, logstash *lstype.Logstash) error {
	var secrets corev1.SecretList
	ns := client.InNamespace(logstash.Namespace)
	matchLabels := NewResourceSelector(logstash.Name)
	if err := c.List(&secrets, ns, matchLabels); err != nil {
		return err
	}

	// Namespace in reference can be empty, in that case we compare it with the namespace of Logstash
	var esRefNamespace string
	if logstash.Spec.ElasticsearchRef.IsDefined() && logstash.Spec.ElasticsearchRef.Namespace != "" {
		esRefNamespace = logstash.Spec.ElasticsearchRef.Namespace
	} else {
		esRefNamespace = logstash.Namespace
	}

	for _, s := range secrets.Items {
		if metav1.IsControlledBy(&s, logstash) || hasBeenCreatedBy(&s, logstash) {
			if !logstash.Spec.ElasticsearchRef.IsDefined() {
				// look for association secrets owned by this logstash instance
				// which should not exist since no ES referenced in the spec
				log.Info("Deleting secret", "namespace", s.Namespace, "secret_name", s.Name, "logstash_name", logstash.Name)
				if err := c.Delete(&s); err != nil && !apierrors.IsNotFound(err) {
					return err
				}
			} else if value, ok := s.Labels[common.TypeLabelName]; ok && value == user.UserType &&
				esRefNamespace != s.Namespace {
				// User secret may live in an other namespace, check if it has changed
				log.Info("Deleting secret", "namespace", s.Namespace, "secretname", s.Name, "logstash_name", logstash.Name)
				if err := c.Delete(&s); err != nil && !apierrors.IsNotFound(err) {
					return err
				}
			}
		}
	}
	return nil
}
