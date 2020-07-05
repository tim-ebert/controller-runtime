package subresource

import "sigs.k8s.io/controller-runtime/pkg/client"

type noOpSubresource struct {}

var _ client.Subresource = &noOpSubresource{}

func (n noOpSubresource) Path() string {
	panic("implement me")
}
