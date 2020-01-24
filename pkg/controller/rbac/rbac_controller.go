package rbac

import (
	"context"
	"regexp"

	"github.com/tektoncd/operator/pkg/flag"

	"github.com/operator-framework/operator-sdk/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8s "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	ctrlLog = logf.Log.WithName("ctrl").WithName("rbac")
)

// Add creates a new RBAC Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	kc, _ := k8s.NewForConfig(mgr.GetConfig())

	return &ReconcileRBAC{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		kc:     kc,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	if _, err := regexp.Compile(flag.IgnorePattern); err != nil {
		ctrlLog.Error(err, "Ignore regex is invalid")
		return err
	}
	// Create a new controller
	c, err := controller.New("rbac-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &corev1.Namespace{}},
		&handler.EnqueueRequestForObject{},
		predicate.GenerationChangedPredicate{},
	)

	return err
}

// blank assignment to verify that ReconcileRBAC implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRBAC{}

// ReconcileRBAC reconciles a PipelineRun object
type ReconcileRBAC struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	kc     *k8s.Clientset
}

func ignoreNotFound(err error) error {
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

// Reconcile reads that state of the cluster for a PipelineRun object and makes changes based on the state read
// and what is in the PipelineRun.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRBAC) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	log := ctrlLog.WithValues("req.name", req.Name)

	if ignore, _ := regexp.MatchString(flag.IgnorePattern, req.Name); ignore {
		return reconcile.Result{}, nil
	}

	log.Info("reconciling rbac sa")

	ns, err := r.getNS(req)
	if err != nil {
		return reconcile.Result{}, ignoreNotFound(err)
	}

	sa, err := r.ensureSA(ns)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.ensureRoleBindings(sa)
	return reconcile.Result{}, err

}

func (r *ReconcileRBAC) getNS(req reconcile.Request) (*corev1.Namespace, error) {
	log := ctrlLog.WithName("ns")

	ns := &corev1.Namespace{}
	if err := r.client.Get(context.TODO(), req.NamespacedName, ns); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "failed to GET namespace")
		}
		return nil, err
	}
	return ns, nil
}

func (r *ReconcileRBAC) ensureSA(ns *corev1.Namespace) (*corev1.ServiceAccount, error) {
	log := ctrlLog.WithName("sa")

	log.Info("finding sa", "sa", flag.PipelineSA, "ns", ns.Name)
	sa := &corev1.ServiceAccount{}
	saType := types.NamespacedName{Name: flag.PipelineSA, Namespace: ns.Name}
	if err := r.client.Get(context.TODO(), saType, sa); err == nil {
		return sa, err
	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	// create sa if not found
	log.Info("creating sa", "sa", flag.PipelineSA, "ns", ns.Name)
	sa = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flag.PipelineSA,
			Namespace: ns.Name,
		},
	}

	err := r.client.Create(context.TODO(), sa)
	return sa, err
}

func (r *ReconcileRBAC) ensureRoleBindings(sa *corev1.ServiceAccount) error {
	log := ctrlLog.WithName("rb").WithValues("ns", sa.Namespace)

	log.Info("finding role-binding edit")
	rbacClient := r.kc.RbacV1()
	editRB, rbErr := rbacClient.RoleBindings(sa.Namespace).Get("edit", metav1.GetOptions{})
	if rbErr != nil && !errors.IsNotFound(rbErr) {
		log.Error(rbErr, "rbac edit get error")
		return rbErr
	}

	log.Info("finding cluster role edit")
	if _, err := rbacClient.ClusterRoles().Get("edit", metav1.GetOptions{}); err != nil {
		log.Error(err, "finding edit cluster role failed")
		return err
	}

	if rbErr != nil && errors.IsNotFound(rbErr) {
		return r.createRoleBinding(sa)
	}

	log.Info("found rbac", "subjects", editRB.Subjects)
	return r.updateRoleBinding(editRB, sa)
}

func (r *ReconcileRBAC) createRoleBinding(sa *corev1.ServiceAccount) error {
	log := ctrlLog.WithName("rb").WithName("new")

	log.Info("create new rolebinding edit")
	rbacClient := r.kc.RbacV1()
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "edit", Namespace: sa.Namespace},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "edit"},
		Subjects:   []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Name: sa.Name, Namespace: sa.Namespace}},
	}

	_, err := rbacClient.RoleBindings(sa.Namespace).Create(rb)
	if err != nil {
		log.Error(err, "creation of edit rb failed")
	}
	return err
}

func (r *ReconcileRBAC) updateRoleBinding(rb *rbacv1.RoleBinding, sa *corev1.ServiceAccount) error {
	log := ctrlLog.WithName("rb").WithName("update")

	subject := rbacv1.Subject{Kind: rbacv1.ServiceAccountKind, Name: sa.Name, Namespace: sa.Namespace}

	if hasSubject(rb.Subjects, subject) {
		log.Info("rolebinding is up to date", "action", "none")
		return nil
	}

	log.Info("update existing rolebinding edit")
	rbacClient := r.kc.RbacV1()
	rb.Subjects = append(rb.Subjects, subject)
	_, err := rbacClient.RoleBindings(sa.Namespace).Update(rb)
	if err != nil {
		log.Error(err, "updation of edit rb failed")
		return err
	}
	log.Error(err, "successfuilly updated edit rb")
	return nil
}

func hasSubject(subjects []rbacv1.Subject, x rbacv1.Subject) bool {
	for _, v := range subjects {
		if v.Name == x.Name && v.Kind == x.Kind && v.Namespace == x.Namespace {
			return true
		}
	}
	return false
}
