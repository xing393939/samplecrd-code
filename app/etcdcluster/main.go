package main

import (
	"flag"
	"k8s.io/client-go/tools/cache"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientset "github.com/xing393939/samplecrd-code/pkg/clients/etcdcluster/clientset/versioned"
	informers "github.com/xing393939/samplecrd-code/pkg/clients/etcdcluster/informers/externalversions"
	"github.com/xing393939/samplecrd-code/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	kubeClient.ServerVersion()

	networkClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	networkInformerFactory := informers.NewSharedInformerFactory(networkClient, time.Second*30)
	networkInformer := networkInformerFactory.Samplecrd().V1().EtcdClusters()
	networkInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) {},
		UpdateFunc: func(old, new interface{}) {},
		DeleteFunc: func(obj interface{}) {},
	})

	go networkInformerFactory.Start(stopCh)

	if err = run(stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func run(stopCh <-chan struct{}) error {
	return nil
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
