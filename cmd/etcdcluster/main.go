package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	samplecrdv1 "github.com/xing393939/samplecrd-code/pkg/apis/etcdcluster/v1"
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

	etcdClusterClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	etcdClusterInformerFactory := informers.NewSharedInformerFactory(etcdClusterClient, time.Second*30)
	etcdClusterInformer := etcdClusterInformerFactory.Samplecrd().V1().EtcdClusters()
	etcdClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cluster := obj.(*samplecrdv1.EtcdCluster)
			createHeadlessServer(kubeClient, cluster)
			createPod(kubeClient, cluster, 0)
		},
		UpdateFunc: func(old, new interface{}) {
			oldCluster := old.(*samplecrdv1.EtcdCluster)
			newCluster := new.(*samplecrdv1.EtcdCluster)
			if oldCluster.ResourceVersion == newCluster.ResourceVersion {
				return
			}
			glog.Info(newCluster.Spec.Size, ",", newCluster.Spec.Version)
		},
		DeleteFunc: func(obj interface{}) {
			cluster := obj.(*samplecrdv1.EtcdCluster)
			err := kubeClient.CoreV1().Pods(cluster.Namespace).DeleteCollection(context.TODO(), metaV1.DeleteOptions{}, metaV1.ListOptions{
				LabelSelector: "clusterName=" + cluster.Name,
			})
			if err != nil {
				glog.Info(err)
			}
			err = kubeClient.CoreV1().Services(cluster.Namespace).Delete(context.TODO(), cluster.Name, metaV1.DeleteOptions{})
			if err != nil {
				glog.Info(err)
			}
		},
	})

	go etcdClusterInformerFactory.Start(stopCh)

	select {}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func createPod(kubeClient *kubernetes.Clientset, cluster *samplecrdv1.EtcdCluster, index int) {
	state := "existing"
	if index == 0 {
		state = "new"
	}
	podName := fmt.Sprintf("%s-%d", cluster.Name, time.Now().UnixNano())
	podEndpoint := "http://" + podName + "." + cluster.Name
	podInitialCluster := podName + "=" + podEndpoint + ":2380"
	commands := fmt.Sprintf("/usr/local/bin/etcd --data-dir=/var/etcd/data --name=%s --initial-advertise-peer-urls=%s "+
		"--listen-peer-urls=%s --listen-client-urls=%s --advertise-client-urls=%s "+
		"--initial-cluster=%s --initial-cluster-state=%s",
		podName, podEndpoint+":2380", "http://0.0.0.0:2380", "http://0.0.0.0:2379", podEndpoint+":2379", podInitialCluster, state)
	if state == "new" {
		commands = fmt.Sprintf("%s --initial-cluster-token=%s", commands, uuid.New())
	}
	pod := v1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   podName,
			Labels: map[string]string{},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:    podName,
				Image:   "ibmcom/etcd:" + cluster.Spec.Version,
				Command: strings.Split(commands, " "),
				Resources: v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("250m"),
						v1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
				Ports: []v1.ContainerPort{
					{
						Name:          "server",
						ContainerPort: int32(2380),
					},
					{
						Name:          "client",
						ContainerPort: int32(2379),
					},
				},
			}},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}
	pod.ObjectMeta.Labels["clusterName"] = cluster.Name
	_, err := kubeClient.CoreV1().Pods(cluster.Namespace).Create(context.TODO(), &pod, metaV1.CreateOptions{})
	if err != nil {
		glog.Info(err)
	}
}

func createHeadlessServer(kubeClient *kubernetes.Clientset, cluster *samplecrdv1.EtcdCluster) {
	service := v1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   cluster.Name,
			Labels: map[string]string{},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "server",
					Port: int32(2380),
				},
				{
					Name: "client",
					Port: int32(2379),
				},
			},
			ClusterIP: "None",
			Selector:  map[string]string{},
		},
	}
	service.Spec.Selector["clusterName"] = cluster.Name
	_, err := kubeClient.CoreV1().Services(cluster.Namespace).Create(context.TODO(), &service, metaV1.CreateOptions{})
	if err != nil {
		glog.Info(err)
	}
}
