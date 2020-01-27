// /*
// Copyright 2017 The Kubernetes Authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	crdappv1alpha1 "k8s.io/sample-controller/pkg/apis/crdappcontroller/v1alpha1"
	clientset "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned"
	crdappscheme "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned/scheme"
	informers "k8s.io/sample-controller/pkg/generated_crdapp/informers/externalversions/crdappcontroller/v1alpha1"
	listers "k8s.io/sample-controller/pkg/generated_crdapp/listers/crdappcontroller/v1alpha1"
)

const underlayControllerAgentName = "underlay-controller"

const (
	// UnderlaySuccessSynced is used as part of the Event 'reason' when a Crdapp is synced
	UnderlaySuccessSynced = "Synced"
	// UnderlayErrResourceExists is used as part of the Event 'reason' when a Crdapp fails
	// to sync due to a Deployment of the same name already existing.
	UnderlayErrResourceExists = "ErrResourceExists"

	// UnderlayMessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	UnderlayMessageResourceExists = "Resource kind %q %q already exists and is not managed by CRD app"
	// UnderlayMessageResourceSynced is the message used for an Event fired when a Crdapp
	// is synced successfully
	UnderlayMessageResourceSynced = "Crd app synced successfully"
)

// UnderlayController is the controller implementation for Crdapp resources
type UnderlayController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// underlayclientset is a clientset for our own API group
	underlayclientset clientset.Interface

	underlaysLister listers.UnderlayLister
	underlaysSynced cache.InformerSynced

	kubeClients   map[string]kubernetes.Interface
	apiextClients map[string]apiextension.Interface
	appClients    map[string]clientset.Interface

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

// NewUnderlayController returns a new CRD app controller
func NewUnderlayController(
	kubeclientset kubernetes.Interface,
	underlayclientset clientset.Interface,
	underlaysInformer informers.UnderlayInformer) *UnderlayController {

	// Create event broadcaster
	// Add crdappcontroller types to the default Kubernetes Scheme so Events can be
	// logged for crdappcontroller types.
	utilruntime.Must(crdappscheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiextscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: underlayControllerAgentName})

	controller := &UnderlayController{
		kubeclientset:     kubeclientset,
		underlayclientset: underlayclientset,
		underlaysLister:   underlaysInformer.Lister(),
		underlaysSynced:   underlaysInformer.Informer().HasSynced,
		kubeClients:       make(map[string]kubernetes.Interface),
		apiextClients:     make(map[string]apiextension.Interface),
		appClients:        make(map[string]clientset.Interface),
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Crdapps"),
		recorder:          recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Crdapp resources change
	underlaysInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueCrdapp,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueCrdapp(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *UnderlayController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Crdapp controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.underlaysSynced); !ok {
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
func (c *UnderlayController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *UnderlayController) processNextWorkItem() bool {
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
func (c *UnderlayController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Crdapp resource with this namespace/name
	underlay, err := c.underlaysLister.Underlays(namespace).Get(name)
	if err != nil {
		// The Crdapp resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("underlay '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	appType := underlay.Spec.Type
	appVersion := underlay.Spec.Version

	// check clients
	_, kok := c.kubeClients[underlay.Name]
	_, aeok := c.apiextClients[underlay.Name]
	_, aok := c.appClients[underlay.Name]
	if !kok || !aeok || !aok {
		kClient, apiextClient, appClient, err := getClients(underlay.Spec.KubeConfig)

		if err != nil {
			return err
		}

		c.kubeClients[underlay.Name] = kClient
		c.apiextClients[underlay.Name] = apiextClient
		c.appClients[underlay.Name] = appClient
	}

	objs, err := getCrdAppConfig(appType, appVersion, c.kubeclientset)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Error during get CRD app config: %s", err))
		return nil
	}

	apps := make([]*crdappv1alpha1.Crdapp, 0, len(objs))

	for _, obj := range objs {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		klog.V(4).Infof("Get kind %s", kind)
		switch kind {
		case "Namespace":
			ns := obj.(*corev1.Namespace)
			klog.V(4).Infof("Get %s %s", kind, ns.Name)
			if _, err = c.createOrUpdateNamespace(underlay, ns); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, ns.Name, err)
				return err
			}
		case "ConfigMap":
			cm := obj.(*corev1.ConfigMap)
			klog.V(4).Infof("Get %s %s", kind, cm.Name)
			if _, err = c.createOrUpdateConfigMap(underlay, cm); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, cm.Name, err)
				return err
			}
		case "CustomResourceDefinition":
			crd := obj.(*apiextensionv1beta1.CustomResourceDefinition)
			klog.V(4).Infof("Get %s %s", kind, crd.Name)
			if _, err = c.createOrUpdateCRD(underlay, crd); err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, crd.Name, err)
				return err
			}
		case "Crdapp":
			crdapp := obj.(*crdappv1alpha1.Crdapp)
			klog.V(4).Infof("Get %s %s", kind, crdapp.Name)
			app, err := c.createOrUpdateCrdapp(underlay, crdapp)
			if err != nil {
				klog.Errorf("Error during process %q %s: %s", kind, crdapp.Name, err)
				return err
			}
			apps = append(apps, app)
		default:
			klog.Errorf("Unknown kind: %s", kind)
		}
	}

	// Finally, we update the status block of the Crdapp resource to reflect the
	// current state of the world
	err = c.updateUnderlayStatus(underlay, apps)
	if err != nil {
		return err
	}

	c.recorder.Event(underlay, corev1.EventTypeNormal, UnderlaySuccessSynced, UnderlayMessageResourceSynced)
	return nil
}

func (c *UnderlayController) updateUnderlayStatus(underlay *crdappv1alpha1.Underlay, crdapp []*crdappv1alpha1.Crdapp) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	var appCount int32 = 0
	for _, app := range crdapp {
		if app.Status.AvailableReplicas > 0 {
			appCount++
		}
	}

	underlayCopy := underlay.DeepCopy()
	underlayCopy.Status.AvailableApps = appCount
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Crdapp resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.underlayclientset.CrdappV1alpha1().Underlays(underlay.Namespace).Update(underlayCopy)
	return err
}

// enqueueCrdapp takes a Crdapp resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Crdapp.
func (c *UnderlayController) enqueueCrdapp(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *UnderlayController) createOrUpdateNamespace(underlay *crdappv1alpha1.Underlay, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	// Get the namespace with the name specified
	ns, err := c.kubeClients[underlay.Name].CoreV1().Namespaces().Get(namespace.Name, metav1.GetOptions{})
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		ns, err = c.kubeClients[underlay.Name].CoreV1().Namespaces().Create(namespace)
	}

	if err != nil {
		return nil, err
	}

	if namespace.Spec.Finalizers != nil &&
		(ns.Spec.Finalizers == nil ||
			len(namespace.Spec.Finalizers) != len(ns.Spec.Finalizers) ||
			namespace.Spec.Finalizers[0] != ns.Spec.Finalizers[0]) {
		klog.V(4).Infof("Underlay %s finalizers: %v, actual namespace spec: %v",
			underlay.Name, namespace.Spec.Finalizers, ns.Spec)
		ns.Spec.Finalizers = namespace.Spec.Finalizers
		ns, err = c.kubeClients[underlay.Name].CoreV1().Namespaces().Update(ns)
	}

	if err != nil {
		return nil, err
	}

	return ns, nil
}

func (c *UnderlayController) createOrUpdateCRD(underlay *crdappv1alpha1.Underlay, customresourcedef *apiextensionv1beta1.CustomResourceDefinition) (*apiextensionv1beta1.CustomResourceDefinition, error) {
	// Get the CRD with the name specified
	crd, err := c.apiextClients[underlay.Name].ApiextensionsV1beta1().CustomResourceDefinitions().Get(customresourcedef.Name, metav1.GetOptions{})
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		klog.Infof("%s not found", customresourcedef.Name)
		crd, err = c.apiextClients[underlay.Name].ApiextensionsV1beta1().CustomResourceDefinitions().Create(customresourcedef)
	}

	if err != nil {
		return nil, err
	}

	return crd, nil
}

func (c *UnderlayController) createOrUpdateConfigMap(underlay *crdappv1alpha1.Underlay, cfgMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	for k, v := range cfgMap.Data {
		rv := strings.ReplaceAll(v, "======", "---")
		cfgMap.Data[k] = rv
	}

	// Get the configmap with the name specified
	cm, err := c.kubeClients[underlay.Name].CoreV1().ConfigMaps(cfgMap.Namespace).Get(cfgMap.Name, metav1.GetOptions{})
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		cm, err = c.kubeClients[underlay.Name].CoreV1().ConfigMaps(cfgMap.Namespace).Create(cfgMap)
	}

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return cm, nil
}

func (c *UnderlayController) createOrUpdateCrdapp(underlay *crdappv1alpha1.Underlay, crdapp *crdappv1alpha1.Crdapp) (*crdappv1alpha1.Crdapp, error) {
	// Get the configmap with the name specified
	app, err := c.appClients[underlay.Name].CrdappV1alpha1().Crdapps(crdapp.Namespace).Get(crdapp.Name, metav1.GetOptions{})
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		app, err = c.appClients[underlay.Name].CrdappV1alpha1().Crdapps(crdapp.Namespace).Create(crdapp)
	}

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	needUpdate := false
	if !strings.EqualFold(crdapp.Spec.Type, app.Spec.Type) {
		klog.V(4).Infof("Underlay %s requires crd app %s with type: %s, actual app type: %s",
			underlay.Name, crdapp.Name, crdapp.Spec.Type, app.Spec.Type)
		app.Spec.Type = crdapp.Spec.Type
		needUpdate = true
	}

	if !strings.EqualFold(crdapp.Spec.Version, app.Spec.Version) {
		klog.V(4).Infof("Underlay %s requires crd app %s with version: %s, actual app version: %s",
			underlay.Name, crdapp.Name, crdapp.Spec.Version, app.Spec.Version)
		app.Spec.Version = crdapp.Spec.Version
		needUpdate = true
	}

	if needUpdate {
		app, err = c.appClients[underlay.Name].CrdappV1alpha1().Crdapps(crdapp.Namespace).Update(app)

		if err != nil {
			return nil, err
		}

	}

	return app, nil
}

func getUnderlayOwnerReference(underlay *crdappv1alpha1.Underlay) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(underlay, crdappv1alpha1.SchemeGroupVersion.WithKind("Underlay")),
	}
}

func getClients(kubeCfgString string) (kubernetes.Interface, apiextension.Interface, clientset.Interface, error) {
	// create a new temp file to hold the kubeconfig
	file, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		klog.Errorf("Failed to create temp file: %s\n", err)
		return nil, nil, nil, err
	}

	kubeCfgByte, err := base64.StdEncoding.DecodeString(kubeCfgString)

	// write the kubeconfig from the secret to the temp file
	_, err = file.Write(kubeCfgByte)
	if err != nil {
		klog.Errorf("Failed to write kubeconfig to temp file: %s\n", err)
		return nil, nil, nil, err
	}
	err = file.Sync()
	if err != nil {
		klog.Errorf("Failed to sync file: %s\n", err)
		return nil, nil, nil, err
	}

	// create the underlay config using the temp file
	kubeCfg, err := clientcmd.BuildConfigFromFlags("", file.Name())
	if err != nil {
		klog.Errorf("Failed to create underlay config from flags: %s\n", err)
		return nil, nil, nil, err
	}

	// create the underlay client using the underlay config
	kubeClient, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		klog.Errorf("Failed to create kubernetes client from config: %s\n", err)
		return nil, nil, nil, err
	}

	apiextClient, err := apiextension.NewForConfig(kubeCfg)
	if err != nil {
		klog.Errorf("Failed to create apiextenrsion client from config: %s\n", err)
		return nil, nil, nil, err
	}

	crdappClient, err := clientset.NewForConfig(kubeCfg)
	if err != nil {
		klog.Errorf("Failed to create app client from config: %s\n", err)
		return nil, nil, nil, err
	}

	return kubeClient, apiextClient, crdappClient, nil
}
