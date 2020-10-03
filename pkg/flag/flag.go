package flag

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

type RuntimeSpec struct {
	Runtime           string
	Version           string
	MinorVersion      string
	SupportedVersions string
}

const (
	// DefaultSA is the default service account
	DefaultSA            = "pipeline"
	DefaultIgnorePattern = "^(openshift|kube)-"

	// DefaultDisableAffinityAssistant is default value of disable affinity assistant flag
	DefaultDisableAffinityAssistant = "true"

	ClusterCRName   = "cluster"
	DefaultTargetNs = "openshift-pipelines"

	// Name of the pipeline controller deployment
	PipelineControllerName = "tekton-pipelines-controller"
	PipelineControllerSA   = "tekton-pipelines-controller"

	PipelineWebhookName          = "tekton-pipelines-webhook"
	PipelineWebhookConfiguration = "webhook.pipeline.tekton.dev"
	SccAnnotationKey             = "operator.tekton.dev"

	// Name of the trigger deployment
	TriggerControllerName       = "tekton-triggers-controller"
	TriggerWebhookName          = "tekton-triggers-webhook"
	TriggerWebhookConfiguration = "webhook.triggers.tekton.dev"

	AnnotationPreserveNS          = "operator.tekton.dev/preserve-namespace"
	AnnotationPreserveRBSubjectNS = "operator.tekton.dev/preserve-rb-subject-namespace"
	LabelProviderType             = "operator.tekton.dev/provider-type"
	ProviderTypeCommunity         = "community"
	ProviderTypeRedHat            = "redhat"
	ProviderTypeCertified         = "certified"

	AnnotationPipelineSupportedVersions = "pipeline.openshift.io/supported-versions"
	LabelPipelineEnvironmentType        = "pipeline.openshift.io/type"
	LabelPipelineRuntime                = "pipeline.openshift.io/runtime"
	LabelPipelineStrategy               = "pipeline.openshift.io/strategy"

	uuidPath     = "deploy/uuid"
	TemplatePath = "deploy/resources/templates"
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
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/jib-maven/0.1/jib-maven.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/maven/0.1/maven.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/tkn/0.1/tkn.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/helm-upgrade-from-source/0.1/helm-upgrade-from-source.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/helm-upgrade-from-repo/0.1/helm-upgrade-from-repo.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/trigger-jenkins-job/0.1/trigger-jenkins-job.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-cli/0.1/git-cli.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/pull-request/0.1/pull-request.yaml",
		"https://raw.githubusercontent.com/tektoncd/catalog/master/task/kubeconfig-creator/0.1/kubeconfig-creator.yaml",
	}

	Runtimes = map[string]RuntimeSpec{
		"s2i-dotnet-3": {Runtime: "dotnet", MinorVersion: "$(params.MINOR_VERSION)", SupportedVersions: "[3.1,3.0]"},
		"s2i-go":       {Runtime: "golang"},
		"s2i-java-8":   {Runtime: "java", SupportedVersions: "[8]"},
		"s2i-java-11":  {Runtime: "java", SupportedVersions: "[11]"},
		"s2i-nodejs":   {Runtime: "nodejs", Version: "$(params.MAJOR_VERSION)", SupportedVersions: "[10,12]"},
		"s2i-perl":     {Runtime: "perl", MinorVersion: "$(params.MINOR_VERSION)", SupportedVersions: "[5.26,5.24]"},
		"s2i-php":      {Runtime: "php", MinorVersion: "$(params.MINOR_VERSION)", SupportedVersions: "[7.2,7.3]"},
		"s2i-python-3": {Runtime: "python", MinorVersion: "$(params.MINOR_VERSION)", SupportedVersions: "[3.6,3.5]"},
		"s2i-ruby":     {Runtime: "ruby", MinorVersion: "$(params.MINOR_VERSION)", SupportedVersions: "[2.5,2.4,2.3]"},
		"buildah":      {},
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

	defaultResDir := filepath.Join("deploy", "resources")
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
