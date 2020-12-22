/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

// Mapper maps an object to a collection of keys to be enqueued.
type Mapper interface {
	// Map maps an object.
	Map(obj client.Object) []reconcile.Request
}

var _ Mapper = MapFunc(nil)

// MapFunc is the signature required for enqueueing requests from a generic function.
// This type is usually used with EnqueueRequestsFromMapFunc when registering an event handler.
// It is a simple implementation of the Mapper interface.
type MapFunc func(client.Object) []reconcile.Request

// Map implements Mapper.
func (f MapFunc) Map(obj client.Object) []reconcile.Request {
	return f(obj)
}

// EnqueueRequestsFromMapFunc enqueues Requests by running a transformation function that outputs a collection
// of reconcile.Requests on each Event.  The reconcile.Requests may be for an arbitrary set of objects
// defined by some user specified transformation of the source Event.  (e.g. trigger Reconciler for a set of objects
// in response to a cluster resize event caused by adding or deleting a Node)
//
// EnqueueRequestsFromMapFunc is frequently used to fan-out updates from one object to one or more other
// objects of a differing type.
//
// For UpdateEvents which contain both a new and old object, the transformation function is run on both
// objects and both sets of Requests are enqueue.
func EnqueueRequestsFromMapFunc(fn MapFunc) EventHandler {
	return &EnqueueRequestsFromMapper{
		Mapper:         fn,
		UpdateBehavior: UpdateWithOldAndNew,
	}
}

var _ EventHandler = &EnqueueRequestsFromMapper{}

// EnqueueRequestsFromMapper enqueues Requests by running a Mapper that outputs a collection
// of reconcile.Requests on each Event.  The reconcile.Requests may be for an arbitrary set of objects
// defined by some user specified transformation of the source Event.  (e.g. trigger Reconciler for a set of objects
// in response to a cluster resize event caused by adding or deleting a Node)
//
// EnqueueRequestsFromMapper is frequently used to fan-out updates from one object to one or more other
// objects of a differing type.
//
// For UpdateEvents, the given UpdateBehaviour decides if only the old, only the new or both objects should be mapped
// and enqueued.
//
// EnqueueRequestsFromMapper can inject fields into the Mapper.
type EnqueueRequestsFromMapper struct {
	// Mapper transforms the argument into a slice of keys to be reconciled
	Mapper Mapper
	// UpdateBehavior decides which object(s) to map and enqueue on updates
	UpdateBehavior UpdateBehavior
}

// Create implements EventHandler
func (e *EnqueueRequestsFromMapper) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, evt.Object)
}

// Update implements EventHandler
func (e *EnqueueRequestsFromMapper) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	switch e.UpdateBehavior {
	case UpdateWithOldAndNew:
		e.mapAndEnqueue(q, evt.ObjectOld)
		e.mapAndEnqueue(q, evt.ObjectNew)
	case UpdateWithOld:
		e.mapAndEnqueue(q, evt.ObjectOld)
	case UpdateWithNew:
		e.mapAndEnqueue(q, evt.ObjectNew)
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestsFromMapper) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, evt.Object)
}

// Generic implements EventHandler
func (e *EnqueueRequestsFromMapper) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, evt.Object)
}

func (e *EnqueueRequestsFromMapper) mapAndEnqueue(q workqueue.RateLimitingInterface, object client.Object) {
	for _, req := range e.Mapper.Map(object) {
		q.Add(req)
	}
}

// InjectFunc implements inject.Injector.
func (e *EnqueueRequestsFromMapper) InjectFunc(f inject.Func) error {
	if f == nil {
		return nil
	}
	return f(e.Mapper)
}

// UpdateBehavior determines how an update should be handled.
type UpdateBehavior uint8

const (
	// UpdateWithOldAndNew considers both, the old as well as the new object, in case of an update.
	UpdateWithOldAndNew UpdateBehavior = iota
	// UpdateWithOld considers only the old object in case of an update.
	UpdateWithOld
	// UpdateWithNew considers only the new object in case of an update.
	UpdateWithNew
)
