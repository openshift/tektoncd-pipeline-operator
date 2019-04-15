package manifestival

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func Parse(pathname string, recursive bool) []unstructured.Unstructured {
	in, out := make(chan []byte, 10), make(chan unstructured.Unstructured, 10)
	go read(pathname, recursive, in)
	go decode(in, out)
	result := []unstructured.Unstructured{}
	for spec := range out {
		result = append(result, spec)
	}
	return result
}

func read(pathname string, recursive bool, sink chan []byte) {
	defer close(sink)
	file, err := os.Stat(pathname)
	if err != nil {
		log.Error(err, "Unable to get file info")
		return
	}
	if file.IsDir() {
		readDir(pathname, recursive, sink)
	} else {
		readFile(pathname, sink)
	}
}

func readDir(pathname string, recursive bool, sink chan []byte) {
	list, err := ioutil.ReadDir(pathname)
	if err != nil {
		log.Error(err, "Unable to read directory")
		return
	}
	for _, f := range list {
		name := path.Join(pathname, f.Name())
		switch {
		case f.IsDir() && recursive:
			readDir(name, recursive, sink)
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
