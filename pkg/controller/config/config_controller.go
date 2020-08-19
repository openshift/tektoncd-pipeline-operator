package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	"github.com/prometheus/common/log"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/flag"
	"github.com/tektoncd/operator/pkg/utils/transform"
	"github.com/tektoncd/operator/pkg/utils/validate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const replaceTimeout = 60

var (
	ctrlLog          = logf.Log.WithName("ctrl").WithName("config")
	recreateResource = mf.Any(mf.ByKind("Deployment"), mf.ByKind("Service"))
	roleBinding      = mf.Any(mf.ByKind("RoleBinding"))
	pipelineVersion  = ""
	triggersVersion  = ""
)

func init() {
	ctrlLog.Info("configuration",
		"resource-watched", flag.ResourceWatched,
		"targetNamespace", flag.TargetNamespace,
		"no-auto-install", flag.NoAutoInstall,
	)
}

// Add creates a new Config Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	rec, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, rec)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	log := ctrlLog.WithName("new-reconciler")
	pipelinePath := filepath.Join(flag.ResourceDir, "pipelines")
	pipeline, err := mf.ManifestFrom(sourceBasedOnRecursion(pipelinePath), mf.UseClient(mfc.NewClient(mgr.GetClient())))
	if err != nil {
		return nil, err
	}

	triggersPath := filepath.Join(flag.ResourceDir, "triggers")
	triggers, err := mf.ManifestFrom(sourceBasedOnRecursion(triggersPath), mf.UseClient(mfc.NewClient(mgr.GetClient())))
	if err != nil {
		return nil, err
	}

	addons, err := readAddons(mgr)
	if err != nil {
		return nil, err
	}

	community, err := fetchCommuntiyResources(mgr)
	if err != nil {
		log.Error(err, "error fetching community resources")
		community = mf.Manifest{}
	}

	return &ReconcileConfig{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		pipeline:  pipeline,
		triggers:  triggers,
		addons:    addons,
		community: community,
	}, nil
}

func fetchCommuntiyResources(mgr manager.Manager) (mf.Manifest, error) {
	if flag.SkipNonRedHatResources {
		return mf.Manifest{}, nil
	}
	//manifestival can take urls/filepaths as input
	//more that one items can be passed as a comma separated list string
	urls := strings.Join(flag.CommunityResourceURLs, ",")
	community, err := mf.ManifestFrom(sourceBasedOnRecursion(urls), mf.UseClient(mfc.NewClient(mgr.GetClient())))
	if err != nil {
		return mf.Manifest{}, err
	}
	return community, nil
}

// this will read all the addons files
func readAddons(mgr manager.Manager) (mf.Manifest, error) {
	// read addons
	addonsPath := filepath.Join(flag.ResourceDir, "addons")
	addons, err := mf.ManifestFrom(sourceBasedOnRecursion(addonsPath), mf.UseClient(mfc.NewClient(mgr.GetClient())))
	if err != nil {
		return mf.Manifest{}, err
	}

	// add optionals to addons if any
	optionalResources, err := readOptional(mgr)
	if err != nil {
		return mf.Manifest{}, err
	}
	addons = addons.Append(optionalResources)
	return addons, nil
}

func readOptional(mgr manager.Manager) (mf.Manifest, error) {
	// check consolesample CRD available
	consoleCRDinstalled, err := validate.CRD(mgr.GetConfig(), "consoleyamlsamples.console.openshift.io")
	if err != nil {
		return mf.Manifest{}, err
	}

	// read optionals only if CRD available
	if consoleCRDinstalled {
		optionalPath := filepath.Join(flag.ResourceDir, "optional")
		client := mfc.NewClient(mgr.GetClient())
		optionalAddons, err := mf.ManifestFrom(sourceBasedOnRecursion(optionalPath), mf.UseClient(client))
		if err != nil {
			return mf.Manifest{}, err
		}
		return optionalAddons, nil
	}
	return mf.Manifest{}, err
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	log := ctrlLog.WithName("add")
	// Create a new controller
	c, err := controller.New("config-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Config
	log.Info("Watching operator config CR")
	err = c.Watch(
		&source.Kind{Type: &op.Config{}},
		&handler.EnqueueRequestForObject{},
		predicate.GenerationChangedPredicate{},
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		&handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &op.Config{},
		})
	if err != nil {
		return err
	}

	if flag.NoAutoInstall {
		return nil
	}

	if err := createCR(mgr.GetClient()); err != nil {
		log.Error(err, "creation of config resource failed")
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileConfig{}

// ReconcileConfig reconciles a Config object
type ReconcileConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	pipeline  mf.Manifest
	triggers  mf.Manifest
	addons    mf.Manifest
	community mf.Manifest
}

// Reconcile reads that state of the cluster for a Config object and makes changes based on the state read
// and what is in the Config.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileConfig) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	log := requestLogger(req, "reconcile")

	log.Info("reconciling config change")

	cfg := &op.Config{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: req.Name}, cfg)

	// ignore all resources except the `resourceWatched`
	if req.Name != flag.ResourceWatched {
		log.Info("ignoring incorrect object")

		// handle resources that are not interesting as error
		if !errors.IsNotFound(err) {
			r.markInvalidResource(cfg)
		}
		return reconcile.Result{}, nil
	}

	// handle deletion of resource
	if errors.IsNotFound(err) {
		// User deleted the cluster resource so delete the pipeine resources
		log.Info("resource has been deleted", "config", cfg.Spec, "status", cfg.Status)
		return r.reconcileDeletion(req, cfg)
	}

	// Error reading the object - requeue the request.
	if err != nil {
		log.Error(err, "requeueing event since there was an error reading object")
		return reconcile.Result{}, err
	}

	log.Info("reconciling at status: " + string(cfg.InstallStatus()))
	switch cfg.InstallStatus() {
	case op.EmptyStatus, op.PipelineApplyError:
		return r.applyPipeline(req, cfg)
	case op.AppliedPipeline, op.PipelineValidateError:
		return r.validatePipeline(req, cfg)
	case op.ValidatedPipeline, op.TriggersError:
		return r.applyTriggers(req, cfg)
	case op.AppliedTriggers, op.TriggersValidateError:
		return r.validateTriggers(req, cfg)
	case op.ValidatedTriggers, op.AddonsError:
		return r.applyAddons(req, cfg)
	case op.AppliedAddons, op.CommunityResourcesError:
		return r.applyCommunityResources(req, cfg)
	case op.InstalledStatus:
		return r.validateVersion(req, cfg)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileConfig) applyPipeline(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "apply-pipeline")

	images := transform.ToLowerCaseKeys(imagesFromEnv(transform.PipelinesImagePrefix))
	newPipeline, err := transformManifest(cfg, &r.pipeline, transform.DeploymentImages(images))
	if err != nil {
		log.Error(err, "failed to apply manifest transformations on pipeline-core")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.PipelineApplyError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}
	r.pipeline = newPipeline

	if err := r.pipeline.Filter(mf.Not(recreateResource)).Apply(); err != nil {
		log.Error(err, "failed to apply non deployment and service pipeline manifest")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.PipelineApplyError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, fmt.Errorf("failed to apply non deployment and service pipeline manifest: %w", err)
	}
	if err := r.pipeline.Filter(recreateResource).Apply(); err != nil {
		if errors.IsInvalid(err) {
			if err := r.deleteAndCreate(); err != nil {
				_ = r.updateStatus(cfg, op.ConfigCondition{
					Code:            op.PipelineApplyError,
					Details:         err.Error(),
					PipelineVersion: pipelineVersion,
					TriggersVersion: triggersVersion,
					Version:         flag.TektonVersion})
				return reconcile.Result{}, fmt.Errorf("failed to recreate pipeline deployments and services: %w", err)
			}
		} else {
			_ = r.updateStatus(cfg, op.ConfigCondition{
				Code:            op.PipelineApplyError,
				Details:         err.Error(),
				PipelineVersion: pipelineVersion,
				TriggersVersion: triggersVersion,
				Version:         flag.TektonVersion})
			return reconcile.Result{}, fmt.Errorf("failed to apply pipeline deployments and service: %w", err)
		}
	}
	log.Info("successfully applied all pipeline resources")

	err = r.updateStatus(cfg, op.ConfigCondition{
		Code:            op.AppliedPipeline,
		PipelineVersion: pipelineVersion,
		TriggersVersion: triggersVersion,
		Version:         flag.TektonVersion,
	})
	return reconcile.Result{Requeue: true}, err
}

func (r *ReconcileConfig) deleteAndCreate() error {
	timeout := time.Duration(replaceTimeout) * time.Second

	propPolicy := mf.PropagationPolicy(metav1.DeletePropagationForeground)
	if err := r.pipeline.Filter(recreateResource).Delete(propPolicy); err != nil {
		log.Error(err, "failed to delete pipeline deployment and service resources")
		return err
	}

	if err := wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		for _, deploy := range r.pipeline.Filter(recreateResource).Resources() {
			if _, err := r.pipeline.Client.Get(&deploy); !apierrors.IsNotFound(err) {
				return false, err
			}
		}
		return true, nil
	}); err != nil {
		return err
	}

	return r.pipeline.Filter(recreateResource).Apply()
}

func (r *ReconcileConfig) validateVersion(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {

	uptoDate := cfg.HasInstalledVersion(flag.TektonVersion) &&
		matchesUUID(cfg.Status.OperatorUUID)

	if !uptoDate {
		return r.applyPipeline(req, cfg)
	}
	// NOTE: do not requeue
	return reconcile.Result{}, nil
}

func matchesUUID(target string) bool {
	uuid := flag.OperatorUUID
	return flag.OperatorUUID == "" || uuid == target
}

func (r *ReconcileConfig) applyTriggers(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "apply-triggers")

	triggerImages := transform.ToLowerCaseKeys(imagesFromEnv(transform.TriggersImagePrefix))
	newTriggers, err := transformManifest(cfg, &r.triggers, transform.DeploymentImages(triggerImages))
	if err != nil {
		log.Error(err, "failed to apply manifest transformations on triggers")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.TriggersError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}
	r.triggers = newTriggers

	if err := r.triggers.Filter(mf.Not(recreateResource)).Apply(); err != nil {
		log.Error(err, "failed to apply non deployment and service trigger manifest")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.TriggersError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, fmt.Errorf("failed to apply non deployment and service trigger manifest: %w", err)
	}
	if err := r.triggers.Filter(recreateResource).Apply(); err != nil {
		if errors.IsInvalid(err) {
			if err := r.deleteAndCreateTriggers(); err != nil {
				_ = r.updateStatus(cfg, op.ConfigCondition{
					Code:            op.TriggersError,
					Details:         err.Error(),
					PipelineVersion: pipelineVersion,
					TriggersVersion: triggersVersion,
					Version:         flag.TektonVersion})
				return reconcile.Result{}, fmt.Errorf("failed to recreate trigger deployments and services: %w", err)
			}
		} else {
			_ = r.updateStatus(cfg, op.ConfigCondition{
				Code:            op.TriggersError,
				Details:         err.Error(),
				PipelineVersion: pipelineVersion,
				TriggersVersion: triggersVersion,
				Version:         flag.TektonVersion})
			return reconcile.Result{}, fmt.Errorf("failed to apply trigger deployments and services: %w", err)
		}
	}
	log.Info("successfully applied all trigger resources")
	err = r.updateStatus(cfg, op.ConfigCondition{
		Code:            op.AppliedTriggers,
		PipelineVersion: pipelineVersion,
		TriggersVersion: triggersVersion,
		Version:         flag.TektonVersion,
	})
	return reconcile.Result{Requeue: true}, err
}

func (r *ReconcileConfig) applyAddons(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "apply-addons")

	//add TaskProviderType label to ClusterTasks (community, redhat, certified)
	addonImages := transform.ToLowerCaseKeys(imagesFromEnv(transform.AddonsImagePrefix))
	addnTfrms := []mf.Transformer{
		transform.InjectLabel(flag.LabelProviderType, flag.ProviderTypeCommunity, transform.Overwrite, "ClusterTask"),
		transform.TaskImages(addonImages),
	}
	newAddons, err := transformManifest(cfg, &r.addons, addnTfrms...)
	if err != nil {
		log.Error(err, "failed to apply manifest transformations on addons")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.AddonsError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}
	r.addons = newAddons

	if err := r.addons.Apply(); err != nil {
		log.Error(err, "failed to apply addons yaml manifest")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.AddonsError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, fmt.Errorf("failed to apply addons yaml manifest: %w", err)
	}

	log.Info("successfully applied all addon resources")

	err = r.updateStatus(cfg, op.ConfigCondition{
		Code:            op.AppliedAddons,
		PipelineVersion: pipelineVersion,
		TriggersVersion: triggersVersion,
		Version:         flag.TektonVersion,
	})
	return reconcile.Result{Requeue: true}, err
}

func (r *ReconcileConfig) deleteAndCreateTriggers() error {
	timeout := time.Duration(replaceTimeout) * time.Second

	propPolicy := mf.PropagationPolicy(metav1.DeletePropagationForeground)
	if err := r.triggers.Filter(recreateResource).Delete(propPolicy); err != nil {
		log.Error(err, "failed to delete triggers deployment and service resources")
		return err
	}

	if err := wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
		for _, deploy := range r.triggers.Filter(recreateResource).Resources() {
			if _, err := r.triggers.Client.Get(&deploy); !apierrors.IsNotFound(err) {
				return false, err
			}
		}
		return true, nil
	}); err != nil {
		return err
	}

	return r.triggers.Filter(recreateResource).Apply()
}

// this will give the component version from the respective controller label
func getComponentVersion(manifest mf.Manifest, controllerName string, labelName string) string {
	labels := manifest.Filter(mf.ByKind("Deployment"), mf.ByName(controllerName)).Resources()[0].GetLabels()
	return labels[labelName]
}

func (r *ReconcileConfig) applyCommunityResources(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "apply-non-redhat-resources")

	//add TaskProviderType label to ClusterTasks (community, redhat, certified)
	addonImages := transform.ToLowerCaseKeys(imagesFromEnv(transform.AddonsImagePrefix))
	addnTfrms := []mf.Transformer{
		// replace kind: Task, with kind: ClusterTask
		transform.ReplaceKind("Task", "ClusterTask"),
		transform.InjectLabel(flag.LabelProviderType, flag.ProviderTypeCommunity, transform.Overwrite),
		transform.TaskImages(addonImages),
	}
	newCommunityResources, err := transformManifest(cfg, &r.community, addnTfrms...)
	if err != nil {
		log.Error(err, "failed to apply manifest transformations on pipeline-addons")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.CommunityResourcesError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}
	r.community = newCommunityResources

	if err := r.community.Apply(); err != nil {
		log.Error(err, "failed to apply non Red Hat resources yaml manifest")
		// ignoring failure to update
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.CommunityResourcesError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}
	log.Info("successfully applied all non Red Hat resources")

	err = r.updateStatus(cfg, op.ConfigCondition{
		Code:            op.InstalledStatus,
		PipelineVersion: pipelineVersion,
		TriggersVersion: triggersVersion,
		Version:         flag.TektonVersion,
	})
	return reconcile.Result{Requeue: true}, err
}

func transformManifest(cfg *op.Config, m *mf.Manifest, addnTfrms ...mf.Transformer) (mf.Manifest, error) {
	rbManifest := m.Filter(roleBinding)
	rest := m.Filter(mf.Not(roleBinding))
	tfs := []mf.Transformer{
		mf.InjectOwner(cfg),
		transform.InjectNamespaceConditional(flag.AnnotationPreserveNS, cfg.Spec.TargetNamespace),
		transform.InjectNamespaceCRDWebhookClientConfig(cfg.Spec.TargetNamespace),
		transform.InjectDefaultSA(flag.DefaultSA),
		transform.SetDisableAffinityAssistant(flag.DefaultDisableAffinityAssistant),
	}

	tfs = append(tfs, addnTfrms...)
	rest, err := rest.Transform(tfs...)
	if err != nil {
		return *m, err
	}

	tfs = []mf.Transformer{
		mf.InjectOwner(cfg),
		transform.InjectNamespaceRoleBindingConditional(flag.AnnotationPreserveNS,
			flag.AnnotationPreserveRBSubjectNS, cfg.Spec.TargetNamespace),
	}
	rbManifest, err = rbManifest.Transform(tfs...)
	if err != nil {
		return *m, err
	}
	return rest.Append(rbManifest), nil
}

func (r *ReconcileConfig) validatePipeline(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "validate-pipeline")
	log.Info("validating pipelines")

	running, err := r.validateDeployments(req, cfg, flag.PipelineControllerName, flag.PipelineWebhookName)
	if err != nil {
		log.Error(err, "failed to validate pipeline controller deployments")
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.PipelineValidateError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}

	if !running {
		return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
	}

	found, err := validate.Webhook(context.TODO(), r.client, flag.PipelineWebhookConfiguration)
	if err != nil {
		log.Error(err, "failed to validate mutating webhook")
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.PipelineValidateError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{RequeueAfter: 15 * time.Second}, err
	}
	if !found {
		return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
	}

	pipelineVersion = getComponentVersion(r.pipeline, flag.PipelineControllerName, "pipeline.tekton.dev/release")
	err = r.updateStatus(cfg, op.ConfigCondition{
		Code:            op.ValidatedPipeline,
		PipelineVersion: pipelineVersion,
		TriggersVersion: triggersVersion,
		Version:         flag.TektonVersion,
	})
	if err != nil {
		return reconcile.Result{}, err

	}
	// requeue with delay for services to be up and running
	return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
}

func (r *ReconcileConfig) validateTriggers(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "validate-triggers")
	log.Info("validating triggers")

	running, err := r.validateDeployments(req, cfg, flag.TriggerControllerName, flag.TriggerWebhookName)
	if err != nil {
		log.Error(err, "failed to validate triggers controller deployments")
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.TriggersValidateError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{}, err
	}

	if !running {
		return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
	}

	found, err := validate.Webhook(context.TODO(), r.client, flag.TriggerWebhookConfiguration)
	if err != nil {
		log.Error(err, "failed to validate mutating webhook")
		_ = r.updateStatus(cfg, op.ConfigCondition{
			Code:            op.TriggersValidateError,
			Details:         err.Error(),
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         flag.TektonVersion})
		return reconcile.Result{RequeueAfter: 15 * time.Second}, err
	}
	if !found {
		return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
	}

	triggersVersion = getComponentVersion(r.triggers, flag.TriggerControllerName, "triggers.tekton.dev/release")
	err = r.updateStatus(cfg, op.ConfigCondition{
		Code:            op.ValidatedTriggers,
		PipelineVersion: pipelineVersion,
		TriggersVersion: triggersVersion,
		Version:         flag.TektonVersion,
	})
	if err != nil {
		return reconcile.Result{}, err

	}
	// requeue with delay for services to be up and running
	return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
}

func (r *ReconcileConfig) validateDeployments(req reconcile.Request, cfg *op.Config, controllerName string, webhookName string) (bool, error) {
	log := requestLogger(req, "validate").WithName("deployments")
	log.Info("validating controllers")

	controller, err := validate.Deployment(context.TODO(),
		r.client,
		controllerName,
		cfg.Spec.TargetNamespace,
	)
	if err != nil {
		log.Error(err, "validating controller deployment error")
		return false, err
	}

	log.Info("validating webhook")
	webhook, err := validate.Deployment(context.TODO(),
		r.client,
		webhookName,
		cfg.Spec.TargetNamespace,
	)
	if err != nil {
		log.Error(err, "validating webhook deployment error")
		return false, err
	}

	if !controller || !webhook {
		log.Info("controller or webhook not yet running")
	}

	return controller && webhook, nil
}

func (r *ReconcileConfig) reconcileDeletion(req reconcile.Request, cfg *op.Config) (reconcile.Result, error) {
	log := requestLogger(req, "delete")

	log.Info("deleting pipeline resources")

	// Requested object not found, could have been deleted after reconcile request.
	// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
	propPolicy := mf.PropagationPolicy(metav1.DeletePropagationForeground)
	if err := r.addons.Delete(propPolicy); err != nil {
		log.Error(err, "failed to delete pipeline addons")
		return reconcile.Result{}, err
	}

	if err := r.pipeline.Delete(propPolicy); err != nil {
		log.Error(err, "failed to delete pipeline core")
		return reconcile.Result{}, err
	}

	// Return and don't requeue
	return reconcile.Result{}, nil
}

// markInvalidResource sets the status of resourse as invalid
func (r *ReconcileConfig) markInvalidResource(cfg *op.Config) {
	err := r.updateStatus(cfg,
		op.ConfigCondition{
			Code:            op.InvalidResource,
			Details:         "metadata.name must be " + flag.ResourceWatched,
			PipelineVersion: pipelineVersion,
			TriggersVersion: triggersVersion,
			Version:         "unknown"})
	if err != nil {
		ctrlLog.Info("failed to update status as invalid")
	}
}

// updateStatus set the status of cfg to s and refreshes cfg to the lastest version
func (r *ReconcileConfig) updateStatus(cfg *op.Config, c op.ConfigCondition) error {

	// NOTE: need to use a deepcopy since Status().Update() seems to reset the
	// APIVersion of the cfg to "" making the object invalid; may be a mechanism
	// to prevent us from using stale version of the object

	tmp := cfg.DeepCopy()
	tmp.Status.OperatorUUID = flag.OperatorUUID
	tmp.Status.Conditions = append([]op.ConfigCondition{c}, tmp.Status.Conditions...)

	if err := r.client.Status().Update(context.TODO(), tmp); err != nil {
		log.Error(err, "status update failed")
		return err
	}

	if err := r.refreshCR(cfg); err != nil {
		log.Error(err, "status update failed to refresh object")
		return err
	}
	return nil
}

func (r *ReconcileConfig) refreshCR(cfg *op.Config) error {
	objKey := types.NamespacedName{
		Namespace: cfg.Namespace,
		Name:      cfg.Name,
	}
	return r.client.Get(context.TODO(), objKey, cfg)
}

func createCR(c client.Client) error {
	log := ctrlLog.WithName("create-cr").WithValues("name", flag.ResourceWatched)
	log.Info("creating a clusterwide resource of config crd")

	cr := &op.Config{
		ObjectMeta: metav1.ObjectMeta{Name: flag.ResourceWatched},
		Spec:       op.ConfigSpec{TargetNamespace: flag.TargetNamespace},
	}

	err := c.Create(context.TODO(), cr)
	if errors.IsAlreadyExists(err) {
		log.Info("skipped creation", "reason", "resource already exists")
		return nil
	}

	return err
}

func requestLogger(req reconcile.Request, context string) logr.Logger {
	return ctrlLog.WithName(context).WithValues(
		"Request.Namespace", req.Namespace,
		"Request.NamespaceName", req.NamespacedName,
		"Request.Name", req.Name)
}

func imagesFromEnv(prefix string) map[string]string {
	images := map[string]string{}
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		keyValue := strings.Split(env, "=")
		name := strings.TrimPrefix(keyValue[0], prefix)
		url := keyValue[1]
		images[name] = url
	}

	return images
}

func sourceBasedOnRecursion(path string) mf.Source {
	if flag.Recursive {
		return mf.Recursive(path)
	}
	return mf.Path(path)
}
