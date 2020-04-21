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
	PipelineWebhookConfiguration = "webhook.pipeline.tekton.dev"
	SccAnnotationKey             = "operator.tekton.dev"

	// Name of the trigger deployment
	TriggerControllerName = "tekton-triggers-controller"
	TriggerWebhookName    = "tekton-triggers-webhook"

	AnnotationPreserveNS  = "operator.tekton.dev/preserve-namespace"
	LabelProviderType     = "operator.tekton.dev/provider-type"
	ProviderTypeCommunity = "community"
	ProviderTypeRedHat    = "redhat"
	ProviderTypeCertified = "certified"

	uuidPath = "deploy/uuid"
)

var (
	flagSet *pflag.FlagSet

	TektonVersion          = "release-next"
	PipelineSA             string
	IgnorePattern          string
	ResourceWatched        string
	ResourceDir            string
	TargetNamespace        string
	NoAutoInstall          bool
	SkipNonRedHatResources bool
	Recursive              bool
	OperatorUUID           string
	CommunityResourceURLs  = []string{
		"https://raw.githubusercontent.com/tektoncd/catalog/master/jib-maven/jib-maven.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/maven/maven.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/tkn/tkn.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/kn/kn.yaml",
	}
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
		&SkipNonRedHatResources, "skip-non-redhat", false,
		"If enabled skip adding Tasks/Pipelines not supported/owned by Red Hat")

	flagSet.BoolVar(
		&Recursive, "recursive", true,
		"If enabled apply manifest file in resource directory recursively")
}
func FlagSet() *pflag.FlagSet {
	return flagSet
}
