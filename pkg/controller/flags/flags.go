package flags

import (
	"flag"
	"path/filepath"
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
	PipelineSA      string
	IgnorePattern   string
	TektonVersion   = "v0.7.0"
	ResourceWatched string
	ResourceDir     string
	TargetNamespace string
	NoAutoInstall   bool
	Recursive       bool
)

func Parse() {
	flag.StringVar(
		&PipelineSA, "rbac-sa", DefaultSA,
		"service account that is auto created; default: "+DefaultSA)
	flag.StringVar(
		&IgnorePattern, "ignore-ns-matching", DefaultIgnorePattern,
		"Namespaces to ignore where SA will be auto-created; default: "+DefaultIgnorePattern)

	flag.StringVar(
		&ResourceWatched, "watch-resource", ClusterCRName,
		"cluster-wide resource that operator honours, default: "+ClusterCRName)

	flag.StringVar(
		&TargetNamespace, "target-namespace", DefaultTargetNs,
		"Namespace where pipeline will be installed default: "+DefaultTargetNs)

	defaultResDir := filepath.Join("deploy", "resources", TektonVersion)
	flag.StringVar(
		&ResourceDir, "resource-dir", defaultResDir,
		"Path to resource manifests, default: "+defaultResDir)

	flag.BoolVar(
		&NoAutoInstall, "no-auto-install", false,
		"Do not automatically install tekton pipelines, default: false")

	flag.BoolVar(
		&Recursive, "recursive", false,
		"If enabled apply manifest file in resource directory recursively")
}
