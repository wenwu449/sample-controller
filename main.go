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
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	crdappclientset "k8s.io/sample-controller/pkg/generated_crdapp/clientset/versioned"
	crdappinformers "k8s.io/sample-controller/pkg/generated_crdapp/informers/externalversions"
	"k8s.io/sample-controller/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	crdappClient, err := crdappclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building crdapp clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	crdappInformerFactory := crdappinformers.NewSharedInformerFactory(crdappClient, time.Second*30)

	controller := NewCrdAppController(kubeClient, crdappClient,
		kubeInformerFactory.Core().V1().Namespaces(),
		kubeInformerFactory.Apps().V1().Deployments(),
		kubeInformerFactory.Core().V1().Secrets(),
		kubeInformerFactory.Rbac().V1().ClusterRoles(),
		kubeInformerFactory.Rbac().V1().ClusterRoleBindings(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		crdappInformerFactory.Crdapp().V1alpha1().Crdapps())

	underlayController := NewUnderlayController(kubeClient, crdappClient,
		crdappInformerFactory.Crdapp().V1alpha1().Underlays())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	crdappInformerFactory.Start(stopCh)

	go func() {
		if err = underlayController.Run(2, stopCh); err != nil {
			klog.Fatalf("Error running underlay controller: %s", err.Error())
		}
	}()

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running app controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
