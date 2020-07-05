package client

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type subresourceClient struct {
	resource    *resourceMeta
	key         ObjectKey
	subresource Subresource
	paramCodec  runtime.ParameterCodec
	err         error
}

var _ SubresourceClient = &subresourceClient{}

func (s *subresourceClient) Get(ctx context.Context, obj runtime.Object) error {
	if s.err != nil {
		return s.err
	}

	r := s.resource
	return r.Get().
		NamespaceIfScoped(s.key.Namespace, r.isNamespaced()).
		Name(s.key.Name).
		Resource(r.resource()).
		SubResource(s.subresource.Path()).
		Do(ctx).
		Into(obj)
}

func (s *subresourceClient) Create(ctx context.Context, obj runtime.Object, opts ...CreateOption) error {
	panic("implement me")
}

func (s *subresourceClient) Delete(ctx context.Context, opts ...DeleteOption) error {
	panic("implement me")
}

func (s *subresourceClient) Update(ctx context.Context, obj runtime.Object, opts ...UpdateOption) error {
	if s.err != nil {
		return s.err
	}

	updateOpts := &UpdateOptions{}
	updateOpts.ApplyOptions(opts)

	r := s.resource
	return r.Put().
		NamespaceIfScoped(s.key.Namespace, r.isNamespaced()).
		Name(s.key.Name).
		Resource(r.resource()).
		SubResource(s.subresource.Path()).
		Body(obj).
		VersionedParams(updateOpts.AsUpdateOptions(), s.paramCodec).
		Do(ctx).
		Into(obj)
}

func (s *subresourceClient) Patch(ctx context.Context, obj runtime.Object, patch Patch, opts ...PatchOption) error {
	panic("implement me")
}

func (s *subresourceClient) Do() error {
	panic("implement me")
}
