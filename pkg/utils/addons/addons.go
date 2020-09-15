package addons

import (
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/flag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type taskGenerator interface {
	generate(pipeline unstructured.Unstructured) (unstructured.Unstructured, error)
}

type generateDeployTask func(interface{}) map[string]interface{}

type pipeline struct {
	environment string
	nameSuffix  string
	generateDeployTask
}

func (p *pipeline) generate(pipeline unstructured.Unstructured) (unstructured.Unstructured, error) {
	newTempRes := unstructured.Unstructured{}
	pipeline.DeepCopyInto(&newTempRes)
	labels := newTempRes.GetLabels()
	labels[flag.LabelPipelineEnvironmentType] = p.environment
	newTempRes.SetLabels(labels)
	name := newTempRes.GetName()
	newTempRes.SetName(name + p.nameSuffix)
	taskDeploy, found, err := unstructured.NestedFieldNoCopy(newTempRes.Object, "spec", "tasks")
	if !found || err != nil {
		return unstructured.Unstructured{}, err
	}
	p.generateDeployTask(taskDeploy)
	return newTempRes, nil
}

func openshiftDeployTask(task interface{}) map[string]interface{} {
	taskDeployMap := task.([]interface{})[2].(map[string]interface{})
	taskDeployMap["taskRef"] = map[string]interface{}{"name": "openshift-client", "kind": "ClusterTask"}
	taskDeployMap["runAfter"] = []interface{}{"build"}
	taskDeployMap["params"] = []interface{}{
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"rollout", "status", "dc/$(params.APP_NAME)"}},
	}
	return taskDeployMap
}

func kubernetesDeployTask(task interface{}) map[string]interface{} {
	taskDeployMap := task.([]interface{})[2].(map[string]interface{})
	taskDeployMap["taskRef"] = map[string]interface{}{"name": "openshift-client", "kind": "ClusterTask"}
	taskDeployMap["runAfter"] = []interface{}{"build"}
	taskDeployMap["params"] = []interface{}{
		map[string]interface{}{"name": "SCRIPT", "value": "kubectl $@"},
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"rollout", "status", "deploy/$(params.APP_NAME)"}},
	}
	return taskDeployMap
}

func knativeDeployTask(task interface{}) map[string]interface{} {
	taskDeployMap := task.([]interface{})[2].(map[string]interface{})
	taskDeployMap["name"] = "kn-service-create"
	taskDeployMap["taskRef"] = map[string]interface{}{"name": "kn", "kind": "ClusterTask"}
	taskDeployMap["runAfter"] = []interface{}{"build"}

	taskDeployMap["params"] = []interface{}{
		map[string]interface{}{"name": "ARGS", "value": []interface{}{"service", "create", "$(params.APP_NAME)", "--image=$(params.IMAGE_NAME)", "--force"}},
	}
	return taskDeployMap
}

func CreatePipelines(template mf.Manifest, client client.Client) (mf.Manifest, error) {
	var pipelines []unstructured.Unstructured

	taskGenerators := []taskGenerator{
		&pipeline{environment: "openshift", nameSuffix: "", generateDeployTask: openshiftDeployTask},
		&pipeline{environment: "kubernetes", nameSuffix: "-deployment", generateDeployTask: kubernetesDeployTask},
		&pipeline{environment: "knative", nameSuffix: "-knative", generateDeployTask: knativeDeployTask},
	}

	for name, spec := range flag.Runtimes {
		newTempRes := unstructured.Unstructured{}
		template.Resources()[0].DeepCopyInto(&newTempRes)
		labels := map[string]string{}
		annotations := map[string]string{}
		if name == "buildah" {
			labels[flag.LabelPipelineStrategy] = "docker"
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
			return mf.Manifest{}, err
		}

		tasks, found, err := unstructured.NestedFieldNoCopy(newTempRes.Object, "spec", "tasks")
		if !found || err != nil {
			return mf.Manifest{}, err
		}
		taskBuild := tasks.([]interface{})[1].(map[string]interface{})
		taskBuild["taskRef"] = map[string]interface{}{"name": name, "kind": "ClusterTask"}
		taskParams, found, err := unstructured.NestedFieldNoCopy(taskBuild, "params")
		if !found || err != nil {
			return mf.Manifest{}, err
		}

		if spec.Version != "" {
			taskParams = append(taskParams.([]interface{}), map[string]interface{}{"name": "VERSION", "value": spec.Version})
			pipelineParams = append(pipelineParams.([]interface{}), map[string]interface{}{"name": "MAJOR_VERSION", "type": "string"})
		}
		if spec.MinorVersion != "" {
			taskParams = append(taskParams.([]interface{}), map[string]interface{}{"name": "MINOR_VERSION", "value": spec.MinorVersion})
			pipelineParams = append(pipelineParams.([]interface{}), map[string]interface{}{"name": "MINOR_VERSION", "type": "string"})
		}
		if err := unstructured.SetNestedField(newTempRes.Object, pipelineParams, "spec", "params"); err != nil {
			return mf.Manifest{}, err
		}

		if err := unstructured.SetNestedField(taskBuild, taskParams, "params"); err != nil {
			return mf.Manifest{}, nil
		}

		//adding the deploy task
		for _, tg := range taskGenerators {
			p, err := tg.generate(newTempRes)
			if err != nil {
				return mf.Manifest{}, err
			}
			pipelines = append(pipelines, p)
		}

	}
	manifests, err := mf.ManifestFrom(mf.Slice(pipelines), mf.UseClient(mfc.NewClient(client)))
	if err != nil {
		return mf.Manifest{}, err
	}
	return manifests, nil
}
