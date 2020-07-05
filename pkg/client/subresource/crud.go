package subresource

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SubresourceClient interface {
	Get(ctx context.Context, obj runtime.Object) error
	Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
	Delete(ctx context.Context, opts ...client.DeleteOption) error
	Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error
}

type crud struct {
	path     string
	resource *client.ResourceMeta
	key      client.ObjectKey
}

func CRUD(path string) client.Subresource {
	return &crud{path: path}
}

func (c *crud) Path() string {
	return c.path
}

func (c *crud) Do(resource *client.ResourceMeta, key client.ObjectKey) error {
	c.resource = resource
	c.key = key
	return nil
}

func (c *crud) Get(ctx context.Context, obj runtime.Object) error {
	return c.resource.Get().
		NamespaceIfScoped(c.key.Namespace, c.resource.IsNamespaced()).
		Name(c.key.Name).
		Resource(c.resource.Resource()).
		SubResource(c.path).
		Do(ctx).
		Into(obj)
}

func (c *crud) Create(ctx context.Context, obj runtime.Object, opts ...CreateOption) error {
	panic("implement me")
}

func (c *crud) Delete(ctx context.Context, opts ...DeleteOption) error {
	panic("implement me")
}

func (c *crud) Update(ctx context.Context, obj runtime.Object, opts ...UpdateOption) error {
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

func (c *crud) Patch(ctx context.Context, obj runtime.Object, patch Patch, opts ...PatchOption) error {
	panic("implement me")
}
