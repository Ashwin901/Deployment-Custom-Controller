package main

import (
	"fmt"

	appsv1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsLister "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientset kubernetes.Interface            // this is required for creating new service for the deployment
	lister    appsLister.DeploymentLister     // used to get the actual object from the cache
	hasSynced cache.InformerSynced            // this would be used to check if the deployment resource was initialised properly in the cache
	queue     workqueue.RateLimitingInterface // adding objects to this queue when an event is detected
}

// function to create a new controller
func newController(clientset kubernetes.Interface, deploymentInformer appsv1.DeploymentInformer) *controller {

	fmt.Println("Creating a new controller")

	controller := &controller{
		clientset: clientset,
		lister:    deploymentInformer.Lister(),
		hasSynced: deploymentInformer.Informer().HasSynced,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "customController"),
	}

	deploymentInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.handleAddEvent,
			DeleteFunc: controller.handleDeleteEvent,
		},
	)

	return controller
}

// starting the controller
func (c *controller) run(ch <-chan struct{}) {

	fmt.Println("Starting the custom controller")

	// checking if the deployment resource was synced
	if !cache.WaitForCacheSync(ch, c.hasSynced) {
		fmt.Print("waiting for cache to be synced\n")
	}

	fmt.Println("Cache synced")
	<-ch // blocking operation
}

// handles the add event for deployment resource
func (c *controller) handleAddEvent(obj interface{}) {
	fmt.Println("A new deployment was added")
}

// handles the delete event for deployment resource
func (c *controller) handleDeleteEvent(obj interface{}) {
	fmt.Println("A deployment was deleted")
}
