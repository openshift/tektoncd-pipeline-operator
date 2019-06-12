package install

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	tektonv1alpha1 "github.com/openshift/tektoncd-pipeline-operator/pkg/apis/tekton/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	mf "github.com/jcrossley3/manifestival"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	tektonVersion = "v0.3.1"
	resourceDir   string
	autoInstall   bool
	recursive     bool
	log           = logf.Log.WithName("controller_install")
)

func init() {
	flag.StringVar(&resourceDir,
		"resource-dir",
		filepath.Join("deploy", "resources", tektonVersion),
		"Path to resource manifests",
	)
	flag.BoolVar(&autoInstall,
		"auto-install",
		false,
		"Automatically create an install custom resource (install pipeline)",
	)
	flag.BoolVar(&recursive,
		"recursive",
		false,
		"If enabled apply manifest file in resource directory recursively",
	)
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Install Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	m, err := mf.NewManifest(resourceDir, recursive, mgr.GetClient())
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, m))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, m mf.Manifest) reconcile.Reconciler {
	return &ReconcileInstall{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		manifest: m,
		config:   mgr.GetConfig(),
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
	err = c.Watch(
		&source.Kind{Type: &tektonv1alpha1.Install{}},
		&handler.EnqueueRequestForObject{},
		predicate.GenerationChangedPredicate{},
	)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tektonv1alpha1.Install{},
	})
	if err != nil {
		return err
	}

	if autoInstall {
		ns, err := k8sutil.GetWatchNamespace()
		if err != nil {
			return err
		}
		go autoCreateCR(mgr.GetClient(), ns)
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcileInstall{}

// ReconcileInstall reconciles a Install object
type ReconcileInstall struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	manifest mf.Manifest
	config   *rest.Config
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
	reqLogger.Info("Pipeline Release Path", "path", resourceDir)

	// Fetch the Install instance
	instance := &tektonv1alpha1.Install{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			err := r.manifest.DeleteAll(
				client.PropagationPolicy(metav1.DeletePropagationForeground),
			)
			if err != nil {
				reqLogger.Error(err, "failed to delete pipeline manifest")
				return reconcile.Result{}, err
			}
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if isUptodate(instance) {
		reqLogger.Info("skipping installation, resources are already installed")
		return reconcile.Result{}, nil
	}

	err = r.install(instance)
	if err != nil {
		reqLogger.Error(err, "failed to apply pipeline manifest")
		return reconcile.Result{}, err
	}

	instance.Status.Version = tektonVersion
	instance.Status.Resources = r.resourceNames()

	err = r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileInstall) install(instance *tektonv1alpha1.Install) error {
	tfs := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
	}
	err := r.manifest.Transform(tfs...)
	if err != nil {
		return err
	}
	err  = r.manifest.ApplyAll()

	for _, path := range instance.Spec.AddOns {
		extension, err := mf.NewManifest(filepath.Join("deploy", "resources", path), true, r.client)
		if err != nil {
			return err
		}
		extension.Transform(tfs...)
		err = extension.ApplyAll()
		if err != nil {
			return err
		}
	}
	return nil
}

func isUptodate(instance *tektonv1alpha1.Install) bool {
	switch {
	case instance.Status.Version != tektonVersion:
		return false
	case instance.Status.Resources != nil:
		return false
	}
	return true
}

func autoCreateCR(c client.Client, ns string) {
	installList := &tektonv1alpha1.InstallList{}
	err := c.List(context.TODO(),
		&client.ListOptions{Namespace: ns},
		installList,
	)
	if err != nil {
		log.Error(err, "unable to list install instances")
	}
	if len(installList.Items) > 0 {
		return
	}

	cr := &tektonv1alpha1.Install{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auto-install",
			Namespace: ns,
		},
	}
	err = c.Create(context.TODO(), cr)
	if err != nil {
		log.Error(err, "unable to create install custom resource")
	}
}

func (r *ReconcileInstall) resourceNames() []string {
	var names []string
	for _, rsc := range r.manifest.Resources {
		name := fmt.Sprintf(
			"%s/%s : %s\n",
			rsc.GetNamespace(),
			rsc.GetName(),
			rsc.GroupVersionKind(),
		)
		names = append(names, name)
	}

	return names
}
