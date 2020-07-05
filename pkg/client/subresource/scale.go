package subresource

import (
	"k8s.io/apimachinery/pkg/api/meta"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type scale struct {
	crud
	restMapper meta.RESTMapper
}

func Scale() client.Subresource {
	return &scale{crud: crud{path: "scale"}}
}

func (s *scale) InjectMapper(mapper meta.RESTMapper) error {
	s.restMapper = mapper
	return nil
}

func (s *scale) Scale(replicas int32) error {
	return nil
}

func (s *scale) CurrentReplicas() (int32, error) {
	return 0, nil
}
