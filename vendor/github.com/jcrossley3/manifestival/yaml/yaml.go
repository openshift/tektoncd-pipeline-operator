package yaml

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	olm = flag.Bool("olm", false,
		"Ignores resources managed by the Operator Lifecycle Manager")
	denamespace = flag.Bool("override-namespace", false,
		"Ignores Namespace resources and creates all others in watched namespace")
	Recursive = flag.Bool("recursive", false,
		"If filename is a directory, process all manifests recursively")
	log = logf.Log.WithName("manifests")
)

func NewYamlManifest(pathname string, config *rest.Config) *YamlManifest {
	client, _ := dynamic.NewForConfig(config)
	log.Info("Reading YAML file", "name", pathname)
	return &YamlManifest{resources: parse(pathname), dynamicClient: client}
}

func (f *YamlManifest) Apply(owner OperandOwner) error {
	for _, spec := range f.resources {
		if !isClusterScoped(spec.GetKind()) {
			// apparently reference counting for cluster-scoped
			// resources is broken, so trust the GC only for ns-scoped
			// dependents
			spec.SetOwnerReferences([]v1.OwnerReference{*v1.NewControllerRef(owner, owner.GroupVersionKind())})
			// overwrite YAML resource to match target
			if *denamespace {
				spec.SetNamespace(owner.GetNamespace())
			}

		}
		c, err := client(spec, f.dynamicClient)
		if err != nil {
			return err
		}
		if _, err = c.Get(spec.GetName(), v1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		} else {
			continue
		}
		log.Info("Creating", "type", spec.GroupVersionKind(), "name", spec.GetName())
		if _, err = c.Create(&spec, v1.CreateOptions{}); err != nil {
			if errors.IsAlreadyExists(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func (f *YamlManifest) Delete() error {
	a := make([]unstructured.Unstructured, len(f.resources))
	copy(a, f.resources)
	// we want to delete in reverse order
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	for _, spec := range a {
		c, err := client(spec, f.dynamicClient)
		if err != nil {
			return err
		}
		if _, err = c.Get(spec.GetName(), v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
		}
		log.Info("Deleting", "type", spec.GroupVersionKind(), "name", spec.GetName())
		if err = c.Delete(spec.GetName(), &v1.DeleteOptions{}); err != nil {
			// ignore GC race conditions triggered by owner references
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (f *YamlManifest) ResourceNames() []string {
	var names []string
	for _, spec := range f.resources {
		names = append(names, fmt.Sprintf("%s (%s)", spec.GetName(), spec.GroupVersionKind()))
	}
	return names
}

type YamlManifest struct {
	dynamicClient dynamic.Interface
	resources     []unstructured.Unstructured
}

type OperandOwner interface {
	v1.Object
	GroupVersionKind() schema.GroupVersionKind
}

func parse(pathname string) []unstructured.Unstructured {
	in, out := make(chan []byte, 10), make(chan unstructured.Unstructured, 10)
	go read(pathname, in)
	go decode(in, out)
	result := []unstructured.Unstructured{}
	for spec := range out {
		if *olm && isManagedByOLM(spec.GetKind()) {
			continue
		}
		if *denamespace && strings.ToLower(spec.GetKind()) == "namespace" {
			continue
		}
		result = append(result, spec)
	}
	return result
}

func read(pathname string, sink chan []byte) {
	defer close(sink)
	file, err := os.Stat(pathname)
	if err != nil {
		log.Error(err, "Unable to get file info")
		return
	}
	if file.IsDir() {
		readDir(pathname, sink)
	} else {
		readFile(pathname, sink)
	}
}

func readDir(pathname string, sink chan []byte) {
	list, err := ioutil.ReadDir(pathname)
	if err != nil {
		log.Error(err, "Unable to read directory")
		return
	}
	for _, f := range list {
		name := path.Join(pathname, f.Name())
		switch {
		case f.IsDir() && *Recursive:
			readDir(name, sink)
		case !f.IsDir():
			readFile(name, sink)
		}
	}
}

func readFile(filename string, sink chan []byte) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err.Error())
	}
	manifests := yaml.NewDocumentDecoder(file)
	defer manifests.Close()
	buf := buffer(file)
	for {
		size, err := manifests.Read(buf)
		if err == io.EOF {
			break
		}
		b := make([]byte, size)
		copy(b, buf)
		sink <- b
	}
}

func decode(in chan []byte, out chan unstructured.Unstructured) {
	for buf := range in {
		spec := unstructured.Unstructured{}
		err := yaml.NewYAMLToJSONDecoder(bytes.NewReader(buf)).Decode(&spec)
		if err != nil {
			if err != io.EOF {
				log.Error(err, "Unable to decode YAML; ignoring")
			}
			continue
		}
		out <- spec
	}
	close(out)
}

func buffer(file *os.File) []byte {
	var size int64 = bytes.MinRead
	if fi, err := file.Stat(); err == nil {
		size = fi.Size()
	}
	return make([]byte, size)
}

func pluralize(kind string) string {
	ret := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(ret, "s"):
		return fmt.Sprintf("%ses", ret)
	case strings.HasSuffix(ret, "policy"):
		return fmt.Sprintf("%sies", ret[:len(ret)-1])
	default:
		return fmt.Sprintf("%ss", ret)
	}
}

func client(spec unstructured.Unstructured, dc dynamic.Interface) (dynamic.ResourceInterface, error) {
	groupVersion, err := schema.ParseGroupVersion(spec.GetAPIVersion())
	if err != nil {
		return nil, err
	}
	groupVersionResource := groupVersion.WithResource(pluralize(spec.GetKind()))
	if ns := spec.GetNamespace(); ns == "" {
		return dc.Resource(groupVersionResource), nil
	} else {
		return dc.Resource(groupVersionResource).Namespace(ns), nil
	}
}

func isClusterScoped(kind string) bool {
	// TODO: something more clever using !APIResource.Namespaced maybe?
	switch strings.ToLower(kind) {
	case "componentstatus",
		"namespace",
		"node",
		"persistentvolume",
		"mutatingwebhookconfiguration",
		"validatingwebhookconfiguration",
		"customresourcedefinition",
		"apiservice",
		"meshpolicy",
		"tokenreview",
		"selfsubjectaccessreview",
		"selfsubjectrulesreview",
		"subjectaccessreview",
		"certificatesigningrequest",
		"podsecuritypolicy",
		"clusterrolebinding",
		"clusterrole",
		"priorityclass",
		"storageclass",
		"volumeattachment":
		return true
	}
	return false
}

func isManagedByOLM(kind string) bool {
	switch strings.ToLower(kind) {
	case "namespace", "role", "rolebinding",
		"clusterrole", "clusterrolebinding",
		"customresourcedefinition", "serviceaccount":
		return true
	}
	return false
}
