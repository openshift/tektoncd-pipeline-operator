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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Set some data in a configmap, only overwriting common keys if they differ
// keep the function here, could be use later.
func UpdateConfigMap(cm *unstructured.Unstructured, data map[string]string, log logr.Logger) {
	for k, v := range data {
		message := []interface{}{"map", cm.GetName(), k, v}
		if x, found, _ := unstructured.NestedFieldNoCopy(cm.Object, "data", k); found {
			if v == x {
				continue
			}
			message = append(message, "previous", x)
		}
		log.Info("Setting", message...)
		unstructured.SetNestedField(cm.Object, v, "data", k)
	}
}
