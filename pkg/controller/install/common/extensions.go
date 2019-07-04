/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package common

import (
	mf "github.com/jcrossley3/manifestival"
	// servingv1alpha1 "github.com/knative/serving-operator/pkg/apis/serving/v1alpha1"
	tektonv1alpha1 "github.com/openshift/tektoncd-pipeline-operator/pkg/apis/tekton/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("common")

type Activities []func(client.Client, *runtime.Scheme, *tektonv1alpha1.Install) (*Extension, error)
type Extender func(*tektonv1alpha1.Install) error
type Extensions []Extension
type Extension struct {
	Transformers []mf.Transformer
	PreInstalls  []Extender
	PostInstalls []Extender
}

func (activities Activities) Extend(c client.Client, scheme *runtime.Scheme, install *tektonv1alpha1.Install) (result Extensions, err error) {
	for _, fn := range activities {
		ext, err := fn(c, scheme, install)
		if err != nil {
			return result, err
		}
		if ext != nil {
			result = append(result, *ext)
		}
	}
	return
}

func (exts Extensions) Transform(instance *tektonv1alpha1.Install) []mf.Transformer {
	result := []mf.Transformer{
		mf.InjectOwner(instance),
		mf.InjectNamespace(instance.GetNamespace()),
	}
	for _, extension := range exts {
		result = append(result, extension.Transformers...)
	}
	// Let any config in instance override everything else
	return append(result, func(u *unstructured.Unstructured) error {

		return nil
	})
}

func (exts Extensions) PreInstall(instance *tektonv1alpha1.Install) error {
	for _, extension := range exts {
		for _, f := range extension.PreInstalls {
			if err := f(instance); err != nil {
				return err
			}
		}
	}
	return nil
}

func (exts Extensions) PostInstall(instance *tektonv1alpha1.Install) error {
	for _, extension := range exts {
		for _, f := range extension.PostInstalls {
			if err := f(instance); err != nil {
				return err
			}
		}
	}
	return nil
}
