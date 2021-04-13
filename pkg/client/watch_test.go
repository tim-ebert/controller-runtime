/*
Copyright 2020 The Kubernetes Authors.

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

package client_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("ClientWithWatch", func() {
	var dep *appsv1.Deployment
	var count uint64 = 0
	var replicaCount int32 = 2
	var ns = "kube-public"
	ctx := context.TODO()

	BeforeEach(func(done Done) {
		atomic.AddUint64(&count, 1)
		dep = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("watch-deployment-name-%v", count), Namespace: ns, Labels: map[string]string{"app": fmt.Sprintf("bar-%v", count)}},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicaCount,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"foo": "bar"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
				},
			},
		}

		var err error
		dep, err = clientset.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		close(done)
	}, serverSideTimeoutSeconds)

	AfterEach(func(done Done) {
		deleteDeployment(ctx, dep, ns)
		close(done)
	}, serverSideTimeoutSeconds)

	Describe("NewWithWatch", func() {
		It("should return a new Client", func(done Done) {
			cl, err := client.NewWithWatch(cfg, client.Options{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cl).NotTo(BeNil())

			close(done)
		})

		watchSuite := func(through client.ObjectList, expectedType client.Object) {
			cl, err := client.NewWithWatch(cfg, client.Options{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cl).NotTo(BeNil())

			watchInterface, err := cl.Watch(ctx, through, &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", dep.Name),
				Namespace:     dep.Namespace,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(watchInterface).NotTo(BeNil())

			defer watchInterface.Stop()

			event, ok := <-watchInterface.ResultChan()
			Expect(ok).To(BeTrue())
			Expect(event.Type).To(BeIdenticalTo(watch.Added))
			Expect(event.Object).To(BeAssignableToTypeOf(expectedType))

			// The metadata client doesn't set GVK so we just use the
			// name and UID as a proxy to confirm that we got the right
			// object.
			metaObject, ok := event.Object.(metav1.Object)
			Expect(ok).To(BeTrue())
			Expect(metaObject.GetName()).To(Equal(dep.Name))
			Expect(metaObject.GetUID()).To(Equal(dep.UID))

		}

		It("should receive a create event when watching the typed object", func(done Done) {
			watchSuite(&appsv1.DeploymentList{}, &appsv1.Deployment{})
			close(done)
		}, 15)

		It("should receive a create event when watching the unstructured object", func(done Done) {
			u := &unstructured.UnstructuredList{}
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "apps",
				Kind:    "Deployment",
				Version: "v1",
			})
			watchSuite(u, &unstructured.Unstructured{})
			close(done)
		}, 15)

		It("should receive a create event when watching the metadata object", func(done Done) {
			m := &metav1.PartialObjectMetadataList{TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}}
			watchSuite(m, &metav1.PartialObjectMetadata{})
			close(done)
		}, 15)
	})

	FIt("should create and watch until condition is met", func(done Done) {
		cl, err := client.NewWithWatch(cfg, client.Options{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cl).NotTo(BeNil())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "watched-cm", Namespace: "default"},
		}
		cmList := &corev1.ConfigMapList{}
		condition := watchtools.ConditionFunc(func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Modified:
			default:
				return false, nil
			}

			switch eventObj := event.Object.(type) {
			case *corev1.ConfigMap:
				status := eventObj.Data["status"]
				fmt.Println("event status=" + status + " resourceVersion=" + eventObj.ResourceVersion)
				return status == "ready", nil
			}
			return false, nil
		})

		_, err = controllerutil.CreateOrUpdate(ctx, cl, cm, func() error {
			cm.Data = make(map[string]string, 1)
			cm.Data["status"] = "ready"
			fmt.Println("create status=" + cm.Data["status"])
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		minResourceVersion := cm.ResourceVersion
		fmt.Println("minResourceVersion=" + minResourceVersion)
		//minResourceVersion = "48"

		fieldSelector := fields.OneTermEqualSelector("metadata.name", cm.GetName())
		lw := &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				if opts.ResourceVersion == "0" {
					opts.ResourceVersion = minResourceVersion
					fmt.Println("setting ListOptions.ResourceVersion=" + minResourceVersion)
				} else {
					fmt.Println("list with ListOptions.ResourceVersion=" + opts.ResourceVersion)
				}

				err := cl.List(ctx, cmList, &client.ListOptions{
					FieldSelector: fieldSelector,
					Namespace:     cm.GetNamespace(),
					Raw:           &opts,
				})
				return cmList, err
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				fmt.Println("watch with ListOptions.ResourceVersion=" + opts.ResourceVersion)
				return cl.Watch(ctx, cmList, &client.ListOptions{
					FieldSelector: fieldSelector,
					Namespace:     cm.GetNamespace(),
					Raw:           &opts,
				})
			},
		}

		go func() {
			defer GinkgoRecover()
			<-time.After(100 * time.Millisecond)
			patch := client.MergeFrom(cm.DeepCopy())
			cm.Data["status"] = "not ready"
			Expect(cl.Patch(ctx, cm, patch)).To(Succeed())
			fmt.Println("patch status=" + cm.Data["status"] + " resourceVersion=" + cm.ResourceVersion)

			<-time.After(100 * time.Millisecond)
			patch = client.MergeFrom(cm.DeepCopy())
			cm.Data["status"] = "ready"
			Expect(cl.Patch(ctx, cm, patch)).To(Succeed())
			fmt.Println("patch status=" + cm.Data["status"] + " resourceVersion=" + cm.ResourceVersion)
		}()

		_, err = watchtools.UntilWithSync(ctx, lw, cm, nil, condition)
		close(done)
	}, 15)

})
