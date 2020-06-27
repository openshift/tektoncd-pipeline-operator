# controller-runtime-client

A [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
implementation of the
[Manifestival](https://github.com/manifestival/manifestival) `Client`.

Usage
-----

```go
import (
    mfc "github.com/manifestival/controller-runtime-client"
    mf  "github.com/manifestival/manifestival"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
    var client client.Client = ...
    
    manifest, err := mfc.NewManifest("dir/", client, mf.Recursive)
    if err != nil {
        panic("Failed to load manifest")
    }
    
    manifest.ApplyAll()
}
```

The `NewManifest` function in this library delegates to the function
of the same name in the `manifestival` package after constructing a
`manifestival.Client` implementation from the `client.Client`.
