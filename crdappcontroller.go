/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	rbacinformers "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	crdappv1alpha1 "k8s.io/sample-controller/pkg/apis/crdappcontroller/v1alpha1"
	clientset "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned"
	crdappscheme "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned/scheme"
	informers "k8s.io/sample-controller/pkg/generated_crdapp/informers/externalversions/crdappcontroller/v1alpha1"
	listers "k8s.io/sample-controller/pkg/generated_crdapp/listers/crdappcontroller/v1alpha1"
)

const crdappControllerAgentName = "crdapp-controller"

const (
	// AppSuccessSynced is used as part of the Event 'reason' when a Crdapp is synced
	AppSuccessSynced = "Synced"
	// AppErrResourceExists is used as part of the Event 'reason' when a Crdapp fails
	// to sync due to a Deployment of the same name already existing.
	AppErrResourceExists = "ErrResourceExists"

	// AppMessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	AppMessageResourceExists = "Resource kind %q %q already exists and is not managed by crd app"
	// AppMessageResourceSynced is the message used for an Event fired when a Crdapp
	// is synced successfully
	AppMessageResourceSynced = "Crd app synced successfully"
)

// CrdappController is the controller implementation for Crdapp resources
type CrdappController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// crdappclientset is a clientset for our own API group
	crdappclientset clientset.Interface

	namespacesLister          corelisters.NamespaceLister
	namespacesSynced          cache.InformerSynced
	deploymentsLister         appslisters.DeploymentLister
	deploymentsSynced         cache.InformerSynced
	secretLister              corelisters.SecretLister
	secretSynced              cache.InformerSynced
	clusterRolesLister        rbaclisters.ClusterRoleLister
	clusterRolesSynced        cache.InformerSynced
	clusterRoleBindingsLister rbaclisters.ClusterRoleBindingLister
	clusterRoleBindingsSynced cache.InformerSynced
	serviceAccountsLister     corelisters.ServiceAccountLister
	serviceAccountsSynced     cache.InformerSynced

	crdappsLister listers.CrdappLister
	crdappsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewCrdAppController returns a new CRD app controller
func NewCrdAppController(
	kubeclientset kubernetes.Interface,
	crdappclientset clientset.Interface,
	namespaceInformer coreinformers.NamespaceInformer,
	deploymentInformer appsinformers.DeploymentInformer,
	secretInformer coreinformers.SecretInformer,
	clusterRoleInformer rbacinformers.ClusterRoleInformer,
	clusterRoleBindingInformer rbacinformers.ClusterRoleBindingInformer,
	serviceAccountInformer coreinformers.ServiceAccountInformer,
	crdappsInformer informers.CrdappInformer) *CrdappController {

	// Create event broadcaster
	// Add crdappcontroller types to the default Kubernetes Scheme so Events can be
	// logged for crdappcontroller types.
	utilruntime.Must(crdappscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: crdappControllerAgentName})

	controller := &CrdappController{
		kubeclientset:             kubeclientset,
		crdappclientset:           crdappclientset,
		namespacesLister:          namespaceInformer.Lister(),
		namespacesSynced:          namespaceInformer.Informer().HasSynced,
		deploymentsLister:         deploymentInformer.Lister(),
		deploymentsSynced:         deploymentInformer.Informer().HasSynced,
		secretLister:              secretInformer.Lister(),
		secretSynced:              secretInformer.Informer().HasSynced,
		clusterRolesLister:        clusterRoleInformer.Lister(),
		clusterRolesSynced:        clusterRoleInformer.Informer().HasSynced,
		clusterRoleBindingsLister: clusterRoleBindingInformer.Lister(),
		clusterRoleBindingsSynced: clusterRoleBindingInformer.Informer().HasSynced,
		serviceAccountsLister:     serviceAccountInformer.Lister(),
		serviceAccountsSynced:     serviceAccountInformer.Informer().HasSynced,
		crdappsLister:             crdappsInformer.Lister(),
		crdappsSynced:             crdappsInformer.Informer().HasSynced,
		workqueue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Crdapps"),
		recorder:                  recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Crdapp resources change
	crdappsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueCrdapp,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueCrdapp(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Crdapp resource will enqueue that Crdapp resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.Secret)
			oldObj := old.(*corev1.Secret)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.ServiceAccount)
			oldObj := old.(*corev1.ServiceAccount)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	clusterRoleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*rbacv1.ClusterRole)
			oldObj := old.(*rbacv1.ClusterRole)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	clusterRoleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*rbacv1.ClusterRoleBinding)
			oldObj := old.(*rbacv1.ClusterRoleBinding)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newObj := new.(*corev1.Namespace)
			oldObj := old.(*corev1.Namespace)
			if newObj.ResourceVersion == oldObj.ResourceVersion {
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *CrdappController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Crdapp controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		c.namespacesSynced,
		c.deploymentsSynced,
		c.clusterRolesSynced,
		c.clusterRoleBindingsSynced,
		c.secretSynced,
		c.serviceAccountsSynced,
		c.crdappsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Crdapp resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *CrdappController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *CrdappController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Crdapp resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Crdapp resource
// with the current status of the resource.
func (c *CrdappController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Crdapp resource with this namespace/name
	crdapp, err := c.crdappsLister.Crdapps(namespace).Get(name)
	if err != nil {
		// The Crdapp resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("crdapp '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	appType := crdapp.Spec.Type
	appVersion := crdapp.Spec.Version

	var deployment *appsv1.Deployment

	objs, err := getCrdAppConfig(appType, appVersion, c.kubeclientset)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Error during get CRD app config: %s", err))
		return nil
	}

	for _, obj := range objs {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		klog.V(4).Infof("Get kind %s", kind)
		switch kind {
		case "Namespace":
			ns := obj.(*corev1.Namespace)
			klog.V(4).Infof("Get %s %s", kind, ns.Name)
			if _, err = c.createOrUpdateNamespace(crdapp, ns); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, ns.Name, err)
				return err
			}
		case "Secret":
			s := obj.(*corev1.Secret)
			klog.V(4).Infof("Get %s %s", kind, s.Name)
			if _, err = c.createOrUpdateSecret(crdapp, s); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, s.Name, err)
				return err
			}
		case "ServiceAccount":
			sa := obj.(*corev1.ServiceAccount)
			klog.V(4).Infof("Get %s %s", kind, sa.Name)
			if _, err = c.createOrUpdateServiceAccount(crdapp, sa); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, sa.Name, err)
				return err
			}
		case "ClusterRole":
			cr := obj.(*rbacv1.ClusterRole)
			klog.V(4).Infof("Get %s %s", kind, cr.Name)
			if _, err = c.createOrUpdateClusterRole(crdapp, cr); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, cr.Name, err)
				return err
			}
		case "ClusterRoleBinding":
			crb := obj.(*rbacv1.ClusterRoleBinding)
			klog.V(4).Infof("Get %s %s", kind, crb.Name)
			if _, err = c.createOrUpdateClusterRoleBinding(crdapp, crb); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, crb.Name, err)
				return err
			}
		case "Deployment":
			deploy := obj.(*appsv1.Deployment)
			klog.V(4).Infof("Get %s %s", kind, deploy.Name)
			if deployment, err = c.createOrUpdateDeployment(crdapp, deploy); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, deploy.Name, err)
				return err
			}
		default:
			klog.Errorf("Unknown kind: %s", kind)
		}
	}

	// Finally, we update the status block of the Crdapp resource to reflect the
	// current state of the world
	if deployment != nil {
		err = c.updateCrdappStatus(crdapp, deployment)
	}
	if err != nil {
		return err
	}

	c.recorder.Event(crdapp, corev1.EventTypeNormal, AppSuccessSynced, AppMessageResourceSynced)
	return nil
}

func (c *CrdappController) updateCrdappStatus(crdapp *crdappv1alpha1.Crdapp, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	appCopy := crdapp.DeepCopy()
	appCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Crdapp resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.crdappclientset.CrdappV1alpha1().Crdapps(crdapp.Namespace).Update(appCopy)
	return err
}

// enqueueCrdapp takes a Crdapp resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Crdapp.
func (c *CrdappController) enqueueCrdapp(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Crdapp resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Crdapp resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *CrdappController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Crdapp, we should not do anything more
		// with it.
		if ownerRef.Kind != "Crdapp" {
			return
		}

		crdapp, err := c.crdappsLister.Crdapps(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("get owner error: '%s'", err)
			klog.V(4).Infof("ignoring orphaned object '%s' of crdapp '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueCrdapp(crdapp)
		return
	}
}

func (c *CrdappController) createOrUpdateNamespace(crdapp *crdappv1alpha1.Crdapp, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	namespace.ObjectMeta.OwnerReferences = getOwnerReference(crdapp)
	// Get the namespace with the name specified
	ns, err := c.namespacesLister.Get(namespace.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		ns, err = c.kubeclientset.CoreV1().Namespaces().Create(namespace)
	}

	if err != nil {
		return nil, err
	}

	if !metav1.IsControlledBy(ns, crdapp) {
		msg := fmt.Sprintf(AppMessageResourceExists, ns.Kind, ns.Name)
		c.recorder.Event(crdapp, corev1.EventTypeWarning, AppErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}

	if namespace.Spec.Finalizers != nil &&
		(ns.Spec.Finalizers == nil ||
			len(namespace.Spec.Finalizers) != len(ns.Spec.Finalizers) ||
			namespace.Spec.Finalizers[0] != ns.Spec.Finalizers[0]) {
		klog.V(4).Infof("Crdapp %s finalizers: %v, actual namespace spec: %v",
			crdapp.Name, namespace.Spec.Finalizers, ns.Spec)
		ns.Spec.Finalizers = namespace.Spec.Finalizers
		ns, err = c.kubeclientset.CoreV1().Namespaces().Update(ns)
	}

	if err != nil {
		return nil, err
	}

	return ns, nil
}

func (c *CrdappController) createOrUpdateSecret(crdapp *crdappv1alpha1.Crdapp, secret *corev1.Secret) (*corev1.Secret, error) {
	secret.ObjectMeta.OwnerReferences = getOwnerReference(crdapp)
	// Get the serect with the namespace/name specified
	s, err := c.secretLister.Secrets(crdapp.Namespace).Get(secret.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		s, err = c.kubeclientset.CoreV1().Secrets(crdapp.Namespace).Create(secret)
	}

	if err != nil {
		return nil, err
	}

	if !metav1.IsControlledBy(s, crdapp) {
		msg := fmt.Sprintf(AppMessageResourceExists, s.Kind, s.Name)
		c.recorder.Event(crdapp, corev1.EventTypeWarning, AppErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}

	return s, nil
}

func (c *CrdappController) createOrUpdateClusterRole(crdapp *crdappv1alpha1.Crdapp, clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	clusterRole.ObjectMeta.OwnerReferences = getOwnerReference(crdapp)
	// Get the clusterrole with the name specified
	cr, err := c.clusterRolesLister.Get(clusterRole.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		cr, err = c.kubeclientset.RbacV1().ClusterRoles().Create(clusterRole)
	}

	if err != nil {
		return nil, err
	}

	if !metav1.IsControlledBy(cr, crdapp) {
		msg := fmt.Sprintf(AppMessageResourceExists, cr.Kind, cr.Name)
		c.recorder.Event(crdapp, corev1.EventTypeWarning, AppErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}

	return cr, nil
}

func (c *CrdappController) createOrUpdateClusterRoleBinding(crdapp *crdappv1alpha1.Crdapp, crBinding *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	crBinding.ObjectMeta.OwnerReferences = getOwnerReference(crdapp)
	// Get the clusterrolebinding with the name specified
	crb, err := c.clusterRoleBindingsLister.Get(crBinding.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		crb, err = c.kubeclientset.RbacV1().ClusterRoleBindings().Create(crBinding)
	}

	if err != nil {
		return nil, err
	}
	if !metav1.IsControlledBy(crb, crdapp) {
		msg := fmt.Sprintf(AppMessageResourceExists, crb.Kind, crb.Name)
		c.recorder.Event(crdapp, corev1.EventTypeWarning, AppErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}
	return crb, nil
}

func (c *CrdappController) createOrUpdateServiceAccount(crdapp *crdappv1alpha1.Crdapp, svcAccount *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	svcAccount.ObjectMeta.OwnerReferences = getOwnerReference(crdapp)
	// Get the serviceaccount with the namespace/name specified
	sa, err := c.serviceAccountsLister.ServiceAccounts(crdapp.Namespace).Get(svcAccount.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		sa, err = c.kubeclientset.CoreV1().ServiceAccounts(crdapp.Namespace).Create(svcAccount)
	}

	if err != nil {
		return nil, err
	}

	if !metav1.IsControlledBy(sa, crdapp) {
		msg := fmt.Sprintf(AppMessageResourceExists, sa.Kind, sa.Name)
		c.recorder.Event(crdapp, corev1.EventTypeWarning, AppErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}
	return sa, nil
}

func (c *CrdappController) createOrUpdateDeployment(crdapp *crdappv1alpha1.Crdapp, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	deployment.ObjectMeta.OwnerReferences = getOwnerReference(crdapp)
	// Get the deployment with the namespace/name specified
	deploy, err := c.deploymentsLister.Deployments(crdapp.Namespace).Get(deployment.Name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deploy, err = c.kubeclientset.AppsV1().Deployments(crdapp.Namespace).Create(deployment)
	}

	if err != nil {
		return nil, err
	}

	if !metav1.IsControlledBy(deploy, crdapp) {
		msg := fmt.Sprintf(AppMessageResourceExists, deploy.Kind, deploy.Name)
		c.recorder.Event(crdapp, corev1.EventTypeWarning, AppErrResourceExists, msg)
		return nil, fmt.Errorf(msg)
	}
	needUpdate := false
	if deployment.Spec.Replicas != nil && *deploy.Spec.Replicas != *deployment.Spec.Replicas {
		klog.V(4).Infof("Crdapp %s requires deployment %s replicas: %d, deployment replicas: %d",
			crdapp.Name, deployment.Name, *deployment.Spec.Replicas, *deploy.Spec.Replicas)
		deploy.Spec.Replicas = deployment.Spec.Replicas
		needUpdate = true
	}

	if !strings.EqualFold(deployment.Spec.Template.Spec.Containers[0].Image, deploy.Spec.Template.Spec.Containers[0].Image) {
		klog.V(4).Infof("Crdapp %s requires deployment %s with image: %s, deployment image: %s",
			crdapp.Name, deployment.Name, deployment.Spec.Template.Spec.Containers[0].Image, deploy.Spec.Template.Spec.Containers[0].Image)
		deploy.Spec.Template.Spec.Containers[0].Image = deployment.Spec.Template.Spec.Containers[0].Image
		needUpdate = true
	}

	if needUpdate {
		deployment, err = c.kubeclientset.AppsV1().Deployments(crdapp.Namespace).Update(deployment)

		if err != nil {
			return nil, err
		}

	}

	return deploy, nil
}

func getOwnerReference(crdapp *crdappv1alpha1.Crdapp) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(crdapp, crdappv1alpha1.SchemeGroupVersion.WithKind("Crdapp")),
	}
}

func getFileContent(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		klog.Fatal(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	return b, err
}

func getFileContentString(filepath string) (string, error) {
	b, err := getFileContent(filepath)
	return string(b), err
}

func getCrdAppConfig(appType, appVersion string, kubeClient kubernetes.Interface) ([]runtime.Object, error) {
	defaultNs := "default"

	cm, err := kubeClient.CoreV1().ConfigMaps(defaultNs).Get(appVersion, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	fileContent, ok := cm.Data[fmt.Sprintf("%s.yaml", appType)]
	if !ok {
		return nil, fmt.Errorf("appType %s config for version %s not found", appType, appVersion)
	}

	sepYamlfiles := strings.Split(fileContent, "---")
	retObjs := make([]runtime.Object, 0, len(sepYamlfiles))
	klog.V(4).Infof("Found %v yamls in %s(%s)", len(sepYamlfiles), appType, appVersion)

	for _, f := range sepYamlfiles {
		if strings.EqualFold(f, "\n") || strings.EqualFold(f, "") {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(f), nil, nil)
		if err != nil {
			klog.Errorf("Error while decoding YAML. Err: %s", err)

			continue
		}

		retObjs = append(retObjs, obj)
	}

	return retObjs, nil
}
