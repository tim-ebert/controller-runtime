package client

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type subresourceClient struct {
	resource    *ResourceMeta
	key         ObjectKey
	subresource Subresource
	paramCodec  runtime.ParameterCodec
	err         error
}

var _ SubresourceClient = &subresourceClient{}


func (s *subresourceClient) Do() error {
	panic("implement me")
}
