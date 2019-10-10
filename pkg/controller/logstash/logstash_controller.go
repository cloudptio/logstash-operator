// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstash

import (
	"reflect"
	"sync/atomic"

	logstashv1beta1 "github.com/elastic/cloud-on-k8s/pkg/apis/logstash/v1beta1"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/annotation"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/association"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/events"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/finalizer"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/keystore"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/operator"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/version"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/watches"
	"github.com/elastic/cloud-on-k8s/pkg/controller/logstash/label"
	"github.com/elastic/cloud-on-k8s/pkg/utils/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	name                = "logstash-controller"
	configChecksumLabel = "logstash.k8s.elastic.co/config-checksum"
)

var log = logf.Log.WithName(name)

// Add creates a new Logstash Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, params operator.Parameters) error {
	reconciler := newReconciler(mgr, params)
	c, err := add(mgr, reconciler)
	if err != nil {
		return err
	}
	return addWatches(c, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, params operator.Parameters) *ReconcileLogstash {
	client := k8s.WrapClient(mgr.GetClient())
	return &ReconcileLogstash{
		Client:         client,
		scheme:         mgr.GetScheme(),
		recorder:       mgr.GetEventRecorderFor(name),
		dynamicWatches: watches.NewDynamicWatches(),
		finalizers:     finalizer.NewHandler(client),
		params:         params,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) (controller.Controller, error) {
	// Create a new controller
	return controller.New(name, mgr, controller.Options{Reconciler: r})
}

func addWatches(c controller.Controller, r *ReconcileLogstash) error {
	// Watch for changes to Logstash
	if err := c.Watch(&source.Kind{Type: &logstashv1beta1.Logstash{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Watch deployments
	if err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &logstashv1beta1.Logstash{},
	}); err != nil {
		return err
	}

	// Watch services
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &logstashv1beta1.Logstash{},
	}); err != nil {
		return err
	}

	// Watch secrets
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &logstashv1beta1.Logstash{},
	}); err != nil {
		return err
	}

	// dynamically watch referenced secrets to connect to Elasticsearch
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, r.dynamicWatches.Secrets); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileLogstash{}

// ReconcileLogstash reconciles a Logstash object
type ReconcileLogstash struct {
	k8s.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder

	finalizers     finalizer.Handler
	dynamicWatches watches.DynamicWatches

	params operator.Parameters

	// iteration is the number of times this controller has run its Reconcile method
	iteration uint64
}

// Reconcile reads that state of the cluster for a Logstash object and makes changes based on the state read and what is
// in the Logstash.Spec
func (r *ReconcileLogstash) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	defer common.LogReconciliationRun(log, request, &r.iteration)()

	// retrieve the logstash object
	var ls logstashv1beta1.Logstash
	if ok, err := association.FetchWithAssociation(r.Client, request, &ls); !ok {
		return reconcile.Result{}, err
	}

	// skip reconciliation if paused
	if common.IsPaused(ls.ObjectMeta) {
		log.Info("Object is paused. Skipping reconciliation", "namespace", ls.Namespace, "logstash_name", ls.Name)
		return common.PauseRequeue, nil
	}

	// check for compatibility with the operator version
	compatible, err := r.isCompatible(&ls)
	if err != nil || !compatible {
		return reconcile.Result{}, err
	}

	// run finalizers
	if err := r.finalizers.Handle(&ls, r.finalizersFor(&ls)...); err != nil {
		if errors.IsConflict(err) {
			// Conflicts are expected and should be resolved on next loop
			log.V(1).Info("Conflict while handling secret watch finalizer", "namespace", ls.Namespace, "logstash_name", ls.Name)
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{}, err
	}

	// Logstash will be deleted nothing to do other than run finalizers
	if ls.IsMarkedForDeletion() {
		return reconcile.Result{}, nil
	}

	// update controller version annotation if necessary
	err = annotation.UpdateControllerVersion(r.Client, &ls, r.params.OperatorInfo.BuildInfo.Version)
	if err != nil {
		return reconcile.Result{}, err
	}

	// main reconciliation logic
	return r.doReconcile(request, &ls)
}

func (r *ReconcileLogstash) isCompatible(ls *logstashv1beta1.Logstash) (bool, error) {
	selector := map[string]string{label.LogstashNameLabelName: ls.Name}
	compat, err := annotation.ReconcileCompatibility(r.Client, ls, selector, r.params.OperatorInfo.BuildInfo.Version)
	if err != nil {
		k8s.EmitErrorEvent(r.recorder, err, ls, events.EventCompatCheckError, "Error during compatibility check: %v", err)
	}

	return compat, err
}

func (r *ReconcileLogstash) doReconcile(request reconcile.Request, ls *logstashv1beta1.Logstash) (reconcile.Result, error) {
	ver, err := version.Parse(ls.Spec.Version)
	if err != nil {
		k8s.EmitErrorEvent(r.recorder, err, ls, events.EventReasonValidation, "Invalid version '%s': %v", ls.Spec.Version, err)
		return reconcile.Result{}, err
	}

	state := NewState(request, ls)
	driver, err := newDriver(r, r.scheme, *ver, r.dynamicWatches, r.recorder)
	if err != nil {
		return reconcile.Result{}, err
	}
	// version specific reconcile
	results := driver.Reconcile(&state, ls, r.params)

	// update status
	err = r.updateStatus(state)
	if err != nil && errors.IsConflict(err) {
		log.V(1).Info("Conflict while updating status", "namespace", ls.Namespace, "logstash_name", ls.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	res, err := results.WithError(err).Aggregate()
	k8s.EmitErrorEvent(r.recorder, err, ls, events.EventReconciliationError, "Reconciliation error: %v", err)
	return res, err
}

func (r *ReconcileLogstash) updateStatus(state State) error {
	current := state.originalLogstash
	if reflect.DeepEqual(current.Status, state.Logstash.Status) {
		return nil
	}
	if state.Logstash.Status.IsDegraded(current.Status) {
		r.recorder.Event(current, corev1.EventTypeWarning, events.EventReasonUnhealthy, "Logstash health degraded")
	}
	log.Info("Updating status", "iteration", atomic.LoadUint64(&r.iteration), "namespace", state.Logstash.Namespace, "logstash_name", state.Logstash.Name)
	return r.Status().Update(state.Logstash)
}

// finalizersFor returns the list of finalizers applying to a given Logstash deployment
func (r *ReconcileLogstash) finalizersFor(ls *logstashv1beta1.Logstash) []finalizer.Finalizer {
	return []finalizer.Finalizer{
		secretWatchFinalizer(*ls, r.dynamicWatches),
		keystore.Finalizer(k8s.ExtractNamespacedName(ls), r.dynamicWatches, ls.Kind),
	}
}
