package main

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/subresource"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
		nil)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		panic(err)
	}

	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-signals.SetupSignalHandler()
		cancel()
	}()

	key := client.ObjectKey{Namespace: "default", Name: "test"}

	deployment := &appsv1.Deployment{}
	if err := c.Get(ctx, key, deployment); err != nil {
		panic(err)
	}

	deploymentJSON, err := json.Marshal(deployment)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deployment: %+v\n", string(deploymentJSON))

	scale := &autoscalingv1.Scale{}
	if err := c.Subresource(deployment, key, subresource.Scale{}).Get(ctx, scale); err != nil {
		panic(err)
	}

	scaleJSON, err := json.Marshal(scale)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Scale: %+v\n", string(scaleJSON))

	if scale.Spec.Replicas > 1 {
		scale.Spec.Replicas = 1
	} else {
		scale.Spec.Replicas = 2
	}

	if err := c.Subresource(deployment, key, subresource.Scale{}).Update(ctx, scale); err != nil {
		panic(err)
	}

	scaleJSON, err = json.Marshal(scale)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Scale: %+v\n", string(scaleJSON))

}
