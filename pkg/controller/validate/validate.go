package validate

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Deployment(ctx context.Context, c client.Client, name, namespace string) (bool, error) {
	dp := &v1.Deployment{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := c.Get(ctx, key, dp)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if dp.Status.AvailableReplicas == *dp.Spec.Replicas {
		return true, nil
	}
	return false, nil
}
