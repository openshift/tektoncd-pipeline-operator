package install

import (
	"context"
	"flag"

	"github.com/jcrossley3/manifestival/yaml"
	tektonv1alpha1 "github.com/openshift-cloud-functions/tektoncd-operator/pkg/apis/tekton/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	filename    string
	autoInstall bool
	log         = logf.Log.WithName("controller_install")
)

func init() {
	flag.StringVar(&filename, "manifest", "deploy/resources",
		"The filename containing the tekton-cd pipeline release resources")
	flag.BoolVar(&autoInstall, "auto-install", false,
		"Automatically install pipeline if none exists")
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Install Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {

	return &ReconcileInstall{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		config: yaml.NewYamlManifest(filename, mgr.GetConfig()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("install-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Install
	err = c.Watch(&source.Kind{Type: &tektonv1alpha1.Install{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Install
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tektonv1alpha1.Install{},
	})
	if err != nil {
		return err
	}

	if autoInstall {
		go createInstallCR(mgr.GetClient())
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcileInstall{}

// ReconcileInstall reconciles a Install object
type ReconcileInstall struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	config *yaml.YamlManifest
}

// Reconcile reads that state of the cluster for a Install object and makes changes based on the state read
// and what is in the Install.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileInstall) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Install")
	reqLogger.Info("Pipeline Release Path", "path", filename)

	// Fetch the Install instance
	instance := &tektonv1alpha1.Install{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.config.Delete()
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.Resources != nil {
		reqLogger.Info("skipping installation resources already set for setup crd")
		return reconcile.Result{}, nil
	}

	if err := r.config.Apply(instance); err != nil {
		reqLogger.Error(err, "failed to apply pipeline manifest")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil

}

func createInstallCR(c client.Client) error {
	installLog := log.WithValues("sub", "auto-install")

	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	installList := &tektonv1alpha1.InstallList{}
	err = c.List(context.TODO(), &client.ListOptions{Namespace: ns}, installList)
	if err != nil {
		installLog.Error(err, "Unable to list Installs")
		return err
	}
	if len(installList.Items) >= 1 {
		return nil
	}

	install := &tektonv1alpha1.Install{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auto-install",
			Namespace: ns,
		},
	}
	if err := c.Create(context.TODO(), install); err != nil {
		installLog.Error(err, "auto-install: failed to create Install CR")
		return err
	}
	return nil
}
