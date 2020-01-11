package flag

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

const (
	// DefaultSA is the default service account
	DefaultSA            = "pipeline"
	DefaultIgnorePattern = "^(openshift|kube)-"

	ClusterCRName   = "cluster"
	DefaultTargetNs = "openshift-pipelines"

	// Name of the pipeline controller deployment
	PipelineControllerName = "tekton-pipelines-controller"
	PipelineControllerSA   = "tekton-pipelines-controller"

	PipelineWebhookName          = "tekton-pipelines-webhook"
	PipelineWebhookConfiguration = "webhook.tekton.dev"
	SccAnnotationKey             = "operator.tekton.dev"

	// Name of the trigger deployment
	TriggerControllerName = "tekton-triggers-controller"
	TriggerWebhookName    = "tekton-triggers-webhook"

	uuidPath = "deploy/uuid"
)

var (
	flagSet *pflag.FlagSet

	TektonVersion   = "release-next"
	PipelineSA      string
	IgnorePattern   string
	ResourceWatched string
	ResourceDir     string
	TargetNamespace string
	NoAutoInstall   bool
	Recursive       bool
	OperatorUUID    string
)

func init() {
	// if the uuid file exists then initialize OperatorUUID else
	// keep the default 0 value ""
	if uuid, err := ioutil.ReadFile(uuidPath); err == nil {
		OperatorUUID = strings.TrimSuffix(string(uuid), "\n")
	}

	flagSet = pflag.NewFlagSet("operator", pflag.ExitOnError)
	flagSet.StringVar(
		&PipelineSA, "rbac-sa", DefaultSA,
		"service account that is auto created; default: "+DefaultSA)
	flagSet.StringVar(
		&IgnorePattern, "ignore-ns-matching", DefaultIgnorePattern,
		"Namespaces to ignore where SA will be auto-created; default: "+DefaultIgnorePattern)

	flagSet.StringVar(
		&ResourceWatched, "watch-resource", ClusterCRName,
		"cluster-wide resource that operator honours, default: "+ClusterCRName)

	flagSet.StringVar(
		&TargetNamespace, "target-namespace", DefaultTargetNs,
		"Namespace where pipeline will be installed default: "+DefaultTargetNs)

	defaultResDir := filepath.Join("deploy", "resources", TektonVersion)
	flagSet.StringVar(
		&ResourceDir, "resource-dir", defaultResDir,
		"Path to resource manifests, default: "+defaultResDir)

	flagSet.BoolVar(
		&NoAutoInstall, "no-auto-install", false,
		"Do not automatically install tekton pipelines, default: false")

	flagSet.BoolVar(
		&Recursive, "recursive", false,
		"If enabled apply manifest file in resource directory recursively")
}

func FlagSet() *pflag.FlagSet {
	return flagSet
}
