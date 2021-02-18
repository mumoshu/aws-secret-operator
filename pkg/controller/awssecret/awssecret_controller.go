package awssecret

import (
	"context"
	"time"

	mumoshuv1alpha1 "github.com/mumoshu/aws-secret-operator/pkg/apis/mumoshu/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	errs "github.com/pkg/errors"
)

var log = logf.Log.WithName("controller_awssecret")

// Add creates a new AWSSecret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAWSSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("awssecret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AWSSecret
	err = c.Watch(&source.Kind{Type: &mumoshuv1alpha1.AWSSecret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// i'm not entirely sure what this section is for, as we don't create any pods.
	// i'm commenting it all out to test functionality without it... -dk
	//	// Watch for changes to secondary resource Pods and requeue the owner AWSSecret
	//	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	//		IsController: true,
	//		OwnerType:    &mumoshuv1alpha1.AWSSecret{},
	//	})
	//	if err != nil {
	//		return err
	//	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAWSSecret{}

// ReconcileAWSSecret reconciles a AWSSecret object
type ReconcileAWSSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	ctx *Context
}

// Reconcile reads that state of the cluster for a AWSSecret object and makes changes based on the state read
// and what is in the AWSSecret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will .
func (r *ReconcileAWSSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the AWSSecret instance
	instance := &mumoshuv1alpha1.AWSSecret{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Secret object
	desired, err := r.newSecretForCR(instance)
	if err != nil {
		return reconcile.Result{}, errs.Wrap(err, "failed to compute secret for cr")
	}

	// Set AWSSecret instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, desired, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Secret already exists
	current := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Secret does not exist, Creating a new Secret", "desired.Namespace", desired.Namespace, "desired.Name", desired.Name)
		err = r.client.Create(context.TODO(), desired)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Secret created successfully - requeue after 5 minutes
		reqLogger.Info("Secret Created successfully, RequeueAfter 5 minutes")
		return reconcile.Result{RequeueAfter: time.Second * 300}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// if Secret exists, only update if versionId has changed
	if string(current.Data["AWSVersionId"]) != desired.StringData["AWSVersionId"] {
		reqLogger.Info("versionId changed, Updating the Secret", "desired.Namespace", desired.Namespace, "desired.Name", desired.Name)
		err = r.client.Update(context.TODO(), desired)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Secret updated successfully - requeue after 5 minutes
		reqLogger.Info("Secret Updated successfully, RequeueAfter 5 minutes")
		return reconcile.Result{RequeueAfter: time.Second * 300}, nil
	}
	return reconcile.Result{RequeueAfter: time.Second * 300}, nil
}

// newSecretForCR returns a Secret with the name/namespace defined in the cr
func (r *ReconcileAWSSecret) newSecretForCR(cr *mumoshuv1alpha1.AWSSecret) (*corev1.Secret, error) {
	labels := map[string]string{
		"app": cr.Name,
	}
	if r.ctx == nil {
		r.ctx = newContext(nil)
	}

	var err error
	stringData := make(map[string]string)
	if cr.Spec.StringDataFrom.SecretsManagerSecretRef.SecretId != "" &&
		cr.Spec.StringDataFrom.SecretsManagerSecretRef.VersionId != "" {
		ref := cr.Spec.StringDataFrom.SecretsManagerSecretRef
		stringData, err = r.ctx.SecretsManagerSecretToKubernetesStringData(ref)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get json secret as map")
		}
	}

	data := make(map[string][]byte)
	if cr.Spec.DataFrom.SecretsManagerSecretRef.SecretId != "" &&
		cr.Spec.DataFrom.SecretsManagerSecretRef.VersionId != "" {
		ref := cr.Spec.DataFrom.SecretsManagerSecretRef
		data, err = r.ctx.SecretsManagerSecretToKubernetesData(ref)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get json secret as map")
		}
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data:       data,
		StringData: stringData,
		Type:       cr.Spec.Type,
	}, nil
}
