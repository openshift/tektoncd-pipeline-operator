package patch

import (
	"bytes"

	jsonpatch "github.com/evanphx/json-patch"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
)

type Patch interface {
	IsRequired() bool
	Merge(*unstructured.Unstructured) error
}

type jsonPatch struct {
	patch  []byte
	config string
}

type strategicPatch struct {
	jsonPatch
	schema strategicpatch.LookupPatchMeta
}

func NewPatch(src, tgt *unstructured.Unstructured) (Patch, error) {
	var original, modified, current []byte
	var err error
	original = getLastAppliedConfig(tgt)
	config := MakeLastAppliedConfig(src)
	if modified, err = src.MarshalJSON(); err != nil {
		return nil, err
	}
	if current, err = tgt.MarshalJSON(); err != nil {
		return nil, err
	}
	obj, err := scheme.Scheme.New(src.GroupVersionKind())
	switch {
	case src.GetKind() == "ConfigMap":
		fallthrough // force "overwrite" merge
	case runtime.IsNotRegisteredError(err):
		return createJsonPatch(original, modified, current, config)
	case err != nil:
		return nil, err
	default:
		return createStrategicPatch(original, modified, current, obj, config)
	}
}

func createJsonPatch(original, modified, current []byte, config string) (*jsonPatch, error) {
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current)
	return &jsonPatch{patch, config}, err
}

func createStrategicPatch(original, modified, current []byte, obj runtime.Object, config string) (*strategicPatch, error) {
	schema, err := strategicpatch.NewPatchMetaFromStruct(obj)
	if err != nil {
		return nil, err
	}
	patch, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, schema, true)
	return &strategicPatch{jsonPatch{patch, config}, schema}, err
}

func (p *jsonPatch) String() string {
	return string(p.patch)
}

func (p *jsonPatch) IsRequired() bool {
	return !bytes.Equal(p.patch, []byte("{}"))
}

func (p *jsonPatch) Merge(spec *unstructured.Unstructured) (err error) {
	var current, result []byte
	if current, err = spec.MarshalJSON(); err != nil {
		return
	}
	if result, err = jsonpatch.MergePatch(current, p.patch); err != nil {
		return
	}
	err = spec.UnmarshalJSON(result)
	if err == nil {
		setLastAppliedConfig(spec, p.config)
	}
	return
}

func (p *strategicPatch) Merge(spec *unstructured.Unstructured) (err error) {
	var current, result []byte
	if current, err = spec.MarshalJSON(); err != nil {
		return
	}
	if result, err = strategicpatch.StrategicMergePatchUsingLookupPatchMeta(current, p.jsonPatch.patch, p.schema); err != nil {
		return
	}
	err = spec.UnmarshalJSON(result)
	if err == nil {
		setLastAppliedConfig(spec, p.config)
	}
	return
}

func getLastAppliedConfig(spec *unstructured.Unstructured) []byte {
	annotations := spec.GetAnnotations()
	if annotations == nil {
		return nil
	}
	return []byte(annotations[v1.LastAppliedConfigAnnotation])
}

func setLastAppliedConfig(spec *unstructured.Unstructured, config string) {
	annotations := spec.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[v1.LastAppliedConfigAnnotation] = config
	spec.SetAnnotations(annotations)
}

func MakeLastAppliedConfig(spec *unstructured.Unstructured) string {
	ann := spec.GetAnnotations()
	if len(ann) > 0 {
		delete(ann, v1.LastAppliedConfigAnnotation)
		spec.SetAnnotations(ann)
	}
	bytes, _ := spec.MarshalJSON()
	return string(bytes)
}
