package main

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsInformers "k8s.io/client-go/informers/apps/v1"
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
func newController(clientset kubernetes.Interface, deploymentInformer appsInformers.DeploymentInformer) *controller {

	fmt.Println("Creating a new controller")

	controller := &controller{
		clientset: clientset,
		lister:    deploymentInformer.Lister(),
		hasSynced: deploymentInformer.Informer().HasSynced,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "customController"),
	}

	// adding handlers for add and delete deployment events
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

// creates new service for the deployment
func (c *controller) createServiceForDeployment(ns, name string) error {
	// get the deployment from lister
	deployment, err := c.lister.Deployments(ns).Get(name)

	if err != nil {
		fmt.Println("Error while getting deployment from lister")
		return err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
			Type:     corev1.ServiceTypeNodePort,
			Selector: getDeploymentLabels(*deployment),
		},
	}

	_, err = c.clientset.CoreV1().Services(ns).Create(context.Background(), svc, metav1.CreateOptions{})

	if err != nil {
		fmt.Println("Error while creating service: ", err.Error())
		return err
	}

	return nil
}

// gets the labels of the pods in the deployment
func getDeploymentLabels(deployment appsv1.Deployment) map[string]string {
	return deployment.Spec.Template.Labels
}

// handles the add event for deployment resource
func (c *controller) handleAddEvent(obj interface{}) {
	fmt.Println("A new deployment was added")
	c.queue.Add(obj)
}

// handles the delete event for deployment resource
func (c *controller) handleDeleteEvent(obj interface{}) {
	fmt.Println("A deployment was deleted")
	c.queue.Add(obj)
}
