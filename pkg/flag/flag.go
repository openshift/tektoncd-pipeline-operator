package flag

import (
	"path/filepath"

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

	// Name of the pipeline webhook deployment
	PipelineWebhookName = "tekton-pipelines-webhook"
	SccAnnotationKey    = "operator.tekton.dev"
)

var (
	flagSet *pflag.FlagSet

	TektonVersion   = "v0.8.0"
	PipelineSA      string
	IgnorePattern   string
	ResourceWatched string
	ResourceDir     string
	TargetNamespace string
	NoAutoInstall   bool
	Recursive       bool
)

func init() {
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
