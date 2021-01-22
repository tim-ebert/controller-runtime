package envtest

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	helperScheme = runtime.NewScheme()
)

// init is required to correctly initialize the helperScheme package variable.
func init() {
	utilruntime.Must(apiextensionsv1.AddToScheme(helperScheme))
	utilruntime.Must(apiextensionsv1beta1.AddToScheme(helperScheme))
	utilruntime.Must(clientgoscheme.AddToScheme(helperScheme))
}

// mergePaths merges two string slices containing paths.
// This function makes no guarantees about order of the merged slice.
func mergePaths(s1, s2 []string) []string {
	m := make(map[string]struct{})
	for _, s := range s1 {
		m[s] = struct{}{}
	}
	for _, s := range s2 {
		m[s] = struct{}{}
	}
	merged := make([]string, len(m))
	i := 0
	for key := range m {
		merged[i] = key
		i++
	}
	return merged
}

// mergeCRDs merges two CRD slices using their names.
// This function makes no guarantees about order of the merged slice.
func mergeCRDs(s1, s2 []client.Object) []client.Object {
	m := make(map[string]*unstructured.Unstructured)
	for _, obj := range runtimeCRDListToUnstructured(s1) {
		m[obj.GetName()] = obj
	}
	for _, obj := range runtimeCRDListToUnstructured(s2) {
		m[obj.GetName()] = obj
	}
	merged := make([]client.Object, len(m))
	i := 0
	for _, obj := range m {
		merged[i] = obj
		i++
	}
	return merged
}

func runtimeCRDListToUnstructured(l []client.Object) []*unstructured.Unstructured {
	res := []*unstructured.Unstructured{}
	for _, obj := range l {
		u := &unstructured.Unstructured{}
		if err := helperScheme.Convert(obj, u, nil); err != nil {
			log.Error(err, "error converting to unstructured object", "object-kind", obj.GetObjectKind())
			continue
		}
		res = append(res, u)
	}
	return res
}

func fillAllTypeMeta(l []client.Object) error {
	for _, o := range l {
		objectGVK := o.GetObjectKind().GroupVersionKind()
		detectedGVK, err := apiutil.GVKForObject(o, helperScheme)
		if err != nil {
			return err
		}

		if objectGVK.Kind == "" {
			objectGVK.Kind = detectedGVK.Kind
		}
		if objectGVK.Group == "" {
			objectGVK.Group = detectedGVK.Group
		}
		if objectGVK.Version == "" {
			objectGVK.Version = detectedGVK.Version
		}
		o.GetObjectKind().SetGroupVersionKind(objectGVK)
	}
	return nil
}
