package subresource

import "sigs.k8s.io/controller-runtime/pkg/client"

type Scale struct{}

var _ client.Subresource = &Scale{}

func (s Scale) Path() string {
	return "scale"
}

func (s Scale) Do() error {
	panic("not implemented")
}
