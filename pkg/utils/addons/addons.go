package addons

import (
	"path"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/flag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type generateDeployTask func(map[string]interface{}) map[string]interface{}
type taskGenerator interface {
	generate(pipeline unstructured.Unstructured, usingPipelineResource bool) (unstructured.Unstructured, error)
}

type pipeline struct {
	environment string
	nameSuffix  string
	generateDeployTask
}

func (p *pipeline) generate(pipeline unstructured.Unstructured, usingPipelineResource bool) (unstructured.Unstructured, error) {
	newTempRes := unstructured.Unstructured{}
	pipeline.DeepCopyInto(&newTempRes)
	labels := newTempRes.GetLabels()
	labels[flag.LabelPipelineEnvironmentType] = p.environment
	newTempRes.SetLabels(labels)
	updatedName := newTempRes.GetName()
	updatedName += p.nameSuffix
	taskDeploy, found, err := unstructured.NestedFieldNoCopy(newTempRes.Object, "spec", "tasks")
	if !found || err != nil {
		return unstructured.Unstructured{}, err
	}

	var index = 2
	if usingPipelineResource {
		index = 1
		updatedName += "-pr"
	}
	newTempRes.SetName(updatedName)

	p.generateDeployTask(taskDeploy.([]interface{})[index].(map[string]interface{}))
	return newTempRes, nil
}

func openshiftDeployTask(deployTask map[string]interface{}) map[string]interface{} {
	deployTask["taskRef"] = map[string]interface{}{"name": "openshift-client", "kind": "ClusterTask"}
	deployTask["runAfter"] = []interface{}{"build"}
	deployTask["params"] = []interface{}{
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"rollout", "status", "dc/$(params.APP_NAME)"}},
	}
	return deployTask
}

func kubernetesDeployTask(deployTask map[string]interface{}) map[string]interface{} {
	deployTask["taskRef"] = map[string]interface{}{"name": "openshift-client", "kind": "ClusterTask"}
	deployTask["runAfter"] = []interface{}{"build"}
	deployTask["params"] = []interface{}{
		map[string]interface{}{"name": "SCRIPT", "value": "kubectl $@"},
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"rollout", "status", "deploy/$(params.APP_NAME)"}},
	}
	return deployTask
}

func knativeDeployTask(deployTask map[string]interface{}) map[string]interface{} {
	deployTask["name"] = "kn-service-create"
	deployTask["taskRef"] = map[string]interface{}{"name": "kn", "kind": "ClusterTask"}
	deployTask["runAfter"] = []interface{}{"build"}
	deployTask["params"] = []interface{}{
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"service", "create", "$(params.APP_NAME)", "--image=$(params.IMAGE_NAME)", "--force"}},
	}
	return deployTask
}

func knativeResourcedDeployTask(deployTask map[string]interface{}) map[string]interface{} {
	deployTask["name"] = "kn-service-create"
	deployTask["taskRef"] = map[string]interface{}{"name": "kn", "kind": "ClusterTask"}
	deployTask["runAfter"] = []interface{}{"build"}
	deployTask["resources"] = map[string]interface{}{
		"inputs": []interface{}{map[string]interface{}{"name": "image", "resource": "app-image", "from": []interface{}{"build"}}},
	}
	deployTask["params"] = []interface{}{
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"service", "create", "$(params.APP_NAME)", "--image=$(resources.inputs.image.url)", "--force"}},
	}
	return deployTask
}

func generateBasePipeline(template mf.Manifest, taskGenerators []taskGenerator, usingPipelineResource bool) ([]unstructured.Unstructured, error) {
	var pipelines []unstructured.Unstructured

	for name, spec := range flag.Runtimes {
		contextParamName := "PATH_CONTEXT"
		newTempRes := unstructured.Unstructured{}
		template.Resources()[0].DeepCopyInto(&newTempRes)
		labels := map[string]string{}
		annotations := map[string]string{}
		if name == "buildah" {
			labels[flag.LabelPipelineStrategy] = "docker"
			contextParamName = "CONTEXT"
		} else {
			labels[flag.LabelPipelineRuntime] = spec.Runtime
		}

		annotations[flag.AnnotationPreserveNS] = "true"
		if spec.SupportedVersions != "" {
			annotations[flag.AnnotationPipelineSupportedVersions] = spec.SupportedVersions
		}
		newTempRes.SetAnnotations(annotations)
		newTempRes.SetLabels(labels)
		newTempRes.SetName(name)
		pipelineParams, found, err := unstructured.NestedFieldNoCopy(newTempRes.Object, "spec", "params")
		if !found || err != nil {
			return nil, err
		}

		tasks, found, err := unstructured.NestedFieldNoCopy(newTempRes.Object, "spec", "tasks")
		if !found || err != nil {
			return nil, err
		}

		taskName := name
		var index = 1
		if usingPipelineResource {
			index = 0
			taskName += "-pr"
		}

		taskBuild := tasks.([]interface{})[index].(map[string]interface{})
		taskBuild["taskRef"] = map[string]interface{}{"name": taskName, "kind": "ClusterTask"}
		taskParams, found, err := unstructured.NestedFieldNoCopy(taskBuild, "params")
		if !found || err != nil {
			return nil, err
		}

		taskParams = append(taskParams.([]interface{}), map[string]interface{}{"name": contextParamName, "value": "$(params.PATH_CONTEXT)"})

		if spec.Version != "" {
			taskParams = append(taskParams.([]interface{}), map[string]interface{}{"name": "VERSION", "value": spec.Version})
			pipelineParams = append(pipelineParams.([]interface{}), map[string]interface{}{"name": "MAJOR_VERSION", "type": "string"})
		}
		if spec.MinorVersion != "" {
			taskParams = append(taskParams.([]interface{}), map[string]interface{}{"name": "MINOR_VERSION", "value": spec.MinorVersion})
			pipelineParams = append(pipelineParams.([]interface{}), map[string]interface{}{"name": "MINOR_VERSION", "type": "string"})
		}

		if err := unstructured.SetNestedField(newTempRes.Object, pipelineParams, "spec", "params"); err != nil {
			return nil, err
		}

		if err := unstructured.SetNestedField(taskBuild, taskParams, "params"); err != nil {
			return nil, nil
		}

		//adding the deploy task
		for _, tg := range taskGenerators {
			p, err := tg.generate(newTempRes, usingPipelineResource)
			if err != nil {
				return nil, err
			}
			pipelines = append(pipelines, p)
		}
	}
	return pipelines, nil
}

func CreatePipelines(templatePath string, client client.Client) (mf.Manifest, error) {
	var pipelines []unstructured.Unstructured
	usingPipelineResource := true
	workspacedTemplate, err := mf.NewManifest(path.Join(templatePath, "pipeline_using_workspace.yaml"))
	if err != nil {
		return mf.Manifest{}, err
	}

	workspacedTaskGenerators := []taskGenerator{
		&pipeline{environment: "openshift", nameSuffix: "", generateDeployTask: openshiftDeployTask},
		&pipeline{environment: "kubernetes", nameSuffix: "-deployment", generateDeployTask: kubernetesDeployTask},
		&pipeline{environment: "knative", nameSuffix: "-knative", generateDeployTask: knativeDeployTask},
	}

	wps, err := generateBasePipeline(workspacedTemplate, workspacedTaskGenerators, !usingPipelineResource)
	if err != nil {
		return mf.Manifest{}, err
	}
	pipelines = append(pipelines, wps...)

	resourcedTemplate, err := mf.NewManifest(path.Join(templatePath, "pipeline_using_resource.yaml"))
	if err != nil {
		return mf.Manifest{}, err
	}

	resourcedTaskGenerators := []taskGenerator{
		&pipeline{environment: "openshift", nameSuffix: "", generateDeployTask: openshiftDeployTask},
		&pipeline{environment: "kubernetes", nameSuffix: "-deployment", generateDeployTask: kubernetesDeployTask},
		&pipeline{environment: "knative", nameSuffix: "-knative", generateDeployTask: knativeResourcedDeployTask},
	}
	rps, err := generateBasePipeline(resourcedTemplate, resourcedTaskGenerators, usingPipelineResource)
	if err != nil {
		return mf.Manifest{}, err
	}
	pipelines = append(pipelines, rps...)

	updatedMf, err := mf.ManifestFrom(mf.Slice(pipelines), mf.UseClient(mfc.NewClient(client)))
	if err != nil {
		return mf.Manifest{}, err
	}
	return updatedMf, nil
}
