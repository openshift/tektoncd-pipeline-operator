package transform

import (
	"fmt"
	"strings"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var transformLog = logf.Log.WithName("transform")

type OverwritePolicy int

const (
	Retain OverwritePolicy = iota
	Overwrite
)

const (
	PipelinesImagePrefix = "IMAGE_PIPELINES_"
	TriggersImagePrefix  = "IMAGE_TRIGGERS_"
	AddonsImagePrefix    = "IMAGE_ADDONS_"

	ArgPrefix   = "arg_"
	ParamPrefix = "param_"
)

// InjectDefaultSA adds default service account into config-defaults configMap
func InjectDefaultSA(defaultSA string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if strings.ToLower(u.GetKind()) != "configmap" {
			return nil
		}
		if u.GetName() != "config-defaults" {
			return nil
		}

		cm := &corev1.ConfigMap{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm)
		if err != nil {
			return err
		}

		cm.Data["default-service-account"] = defaultSA
		unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cm)
		if err != nil {
			return err
		}

		u.SetUnstructuredContent(unstrObj)
		return nil
	}
}

func InjectNamespaceConditional(preserveNamespace, targetNamespace string) mf.Transformer {
	tf := mf.InjectNamespace(targetNamespace)
	return func(u *unstructured.Unstructured) error {
		annotations := u.GetAnnotations()
		val, ok := annotations[preserveNamespace]
		if ok && val == "true" {
			return nil
		}
		return tf(u)
	}
}

func InjectNamespaceRoleBindingSubjects(targetNamespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := strings.ToLower(u.GetKind())
		if kind != "rolebinding" {
			return nil
		}
		subjects, found, err := unstructured.NestedFieldNoCopy(u.Object, "subjects")
		if !found || err != nil {
			return err
		}
		for _, subject := range subjects.([]interface{}) {
			m := subject.(map[string]interface{})
			if _, ok := m["namespace"]; ok {
				m["namespace"] = targetNamespace
			}
		}
		return nil
	}
}

func InjectNamespaceCRDWebhookClientConfig(targetNamespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := strings.ToLower(u.GetKind())
		if kind != "customresourcedefinition" {
			return nil
		}
		service, found, err := unstructured.NestedFieldNoCopy(u.Object, "spec", "conversion", "webhookClientConfig", "service")
		if !found || err != nil {
			return err
		}
		m := service.(map[string]interface{})
		if _, ok := m["namespace"]; ok {
			m["namespace"] = targetNamespace
		}
		return nil
	}
}

func ReplaceKind(fromKind, toKind string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := u.GetKind()

		if kind != fromKind {
			return nil
		}
		err := unstructured.SetNestedField(u.Object, toKind, "kind")
		if err != nil {
			return fmt.Errorf(
				"failed to change resource Name:%s, KIND from %s to %s, %s",
				u.GetName(),
				fromKind,
				toKind,
				err,
			)
		}
		return nil
	}
}

//InjectLabel adds label key:value to a resource
// overwritePolicy (Retain/Overwrite) decides whehther to overwrite an already existing label
// []kinds specify the Kinds on which the label should be applied
// if len(kinds) = 0, label will be apllied to all/any resources irrespective of its Kind
func InjectLabel(key, value string, overwritePolicy OverwritePolicy, kinds ...string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := u.GetKind()
		if len(kinds) != 0 && !ItemInSlice(kind, kinds) {
			return nil
		}
		labels, found, err := unstructured.NestedStringMap(u.Object, "metadata", "labels")
		if err != nil {
			return fmt.Errorf("could not find labels set, %q", err)
		}
		if overwritePolicy == Retain && found {
			if _, ok := labels[key]; ok {
				return nil
			}
		}
		if !found {
			labels = map[string]string{}
		}
		labels[key] = value
		err = unstructured.SetNestedStringMap(u.Object, labels, "metadata", "labels")
		if err != nil {
			return fmt.Errorf("error updating labels for %s:%s, %s", kind, u.GetName(), err)
		}
		return nil
	}
}

func DeploymentImages(images map[string]string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" {
			return nil
		}

		d := &appsv1.Deployment{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, d)
		if err != nil {
			return err
		}

		containers := d.Spec.Template.Spec.Containers
		replaceContainerImages(containers, images)

		unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(d)
		if err != nil {
			return err
		}
		u.SetUnstructuredContent(unstrObj)

		return nil
	}
}

func replaceContainerImages(containers []corev1.Container, images map[string]string) {
	for i, container := range containers {
		name := formKey("", container.Name)
		if url, exist := images[name]; exist {
			containers[i].Image = url
		}

		replaceContainersArgsImage(&container, images)
	}
}

func replaceContainersArgsImage(container *corev1.Container, images map[string]string) {
	for a, arg := range container.Args {
		if argVal, hasArg := splitsByEqual(arg); hasArg {
			argument := formKey(ArgPrefix, argVal[0])
			if url, exist := images[argument]; exist {
				container.Args[a] = argVal[0] + "=" + url
			}
			continue
		}

		argument := formKey(ArgPrefix, arg)
		if url, exist := images[argument]; exist {
			container.Args[a+1] = url
		}
	}

}

func formKey(prefix, arg string) string {
	argument := strings.ToLower(arg)
	if prefix != "" {
		argument = prefix + argument
	}
	return strings.ReplaceAll(argument, "-", "_")
}

func splitsByEqual(arg string) ([]string, bool) {
	values := strings.Split(arg, "=")
	if len(values) == 2 {
		return values, true
	}

	return values, false
}

func TaskImages(images map[string]string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "ClusterTask" {
			return nil
		}

		steps, found, err := unstructured.NestedSlice(u.Object, "spec", "steps")
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		replaceStepsImages(steps, images)
		err = unstructured.SetNestedField(u.Object, steps, "spec", "steps")
		if err != nil {
			return err
		}

		params, found, err := unstructured.NestedSlice(u.Object, "spec", "params")
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		replaceParamsImage(params, images)
		err = unstructured.SetNestedField(u.Object, params, "spec", "params")
		if err != nil {
			return err
		}
		return nil
	}
}

func replaceStepsImages(steps []interface{}, override map[string]string) {
	for _, s := range steps {
		step := s.(map[string]interface{})
		name, ok := step["name"].(string)
		if !ok {
			transformLog.Info("Unable to get the step", "step", s)
			continue
		}

		name = formKey("", name)
		image, found := override[name]
		if !found || image == "" {
			transformLog.Info("Image not found", "step", name, "action", "skip")
			continue
		}
		step["image"] = image
	}
}

func replaceParamsImage(params []interface{}, override map[string]string) {
	for _, p := range params {
		param := p.(map[string]interface{})
		name, ok := param["name"].(string)
		if !ok {
			transformLog.Info("Unable to get the pram", "param", p)
			continue
		}

		name = formKey(ParamPrefix, name)
		image, found := override[name]
		if !found || image == "" {
			transformLog.Info("Image not found", "step", name, "action", "skip")
			continue
		}
		param["default"] = image
	}
}

func ItemInSlice(item string, items []string) bool {
	for _, v := range items {
		if v == item {
			return true
		}
	}
	return false
}

func ToLowerCaseKeys(keyValues map[string]string) map[string]string {
	newMap := map[string]string{}

	for k, v := range keyValues {
		key := strings.ToLower(k)
		newMap[key] = v
	}

	return newMap
}
