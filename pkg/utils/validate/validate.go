package validate

import (
	"context"
	"fmt"

	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/apps/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	v1Options "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ignoreNotFound(err error) error {
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func Deployment(ctx context.Context, c client.Client, name, namespace string) (bool, error) {
	deployment := v1.Deployment{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	err := c.Get(ctx, key, &deployment)
	if err != nil {
		return false, ignoreNotFound(err)
	}

	if deployment.Generation <= deployment.Status.ObservedGeneration {
		cond := getDeploymentCondition(deployment.Status, v1.DeploymentProgressing)
		if cond != nil && cond.Reason == "ProgressDeadlineExceeded" {
			return false, fmt.Errorf("deployment %q exceeded its progress deadline", deployment.Name)
		}
		if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
			return false, nil
		}
		if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
			return false, nil
		}
		if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

func getDeploymentCondition(status v1.DeploymentStatus, condType v1.DeploymentConditionType) *v1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

func Webhook(ctx context.Context, c client.Client, name string) (bool, error) {
	webhook := admissionregistration.MutatingWebhookConfiguration{}
	key := client.ObjectKey{Name: name}
	err := c.Get(ctx, key, &webhook)
	return err == nil, ignoreNotFound(err)
}

func CRD(config *rest.Config, crdName string) (bool, error) {
	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return false, err
	}
	_, err = apiextensionsclientset.ApiextensionsV1beta1().
		CustomResourceDefinitions().Get(crdName, v1Options.GetOptions{})
	return err == nil, ignoreNotFound(err)
}
