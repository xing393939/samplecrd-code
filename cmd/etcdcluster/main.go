package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
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
	kubeClient.ServerVersion()

	etcdClusterClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	etcdClusterInformerFactory := informers.NewSharedInformerFactory(etcdClusterClient, time.Second*30)
	etcdClusterInformer := etcdClusterInformerFactory.Samplecrd().V1().EtcdClusters()
	etcdClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cluster := obj.(*samplecrdv1.EtcdCluster)
			createPod(kubeClient, cluster, 0)
		},
		UpdateFunc: func(old, new interface{}) {
			oldEtcdCluster := old.(*samplecrdv1.EtcdCluster)
			newEtcdCluster := new.(*samplecrdv1.EtcdCluster)
			if oldEtcdCluster.ResourceVersion == newEtcdCluster.ResourceVersion {
				return
			}
			glog.Info(newEtcdCluster.Spec.Size, ",", newEtcdCluster.Spec.Version)
		},
		DeleteFunc: func(obj interface{}) {
			cluster := obj.(*samplecrdv1.EtcdCluster)
			err := kubeClient.CoreV1().Pods(cluster.Namespace).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
				LabelSelector: "clusterName=" + cluster.Name,
			})
			if err != nil {
				glog.Fatal(err)
			}
		},
	})

	go etcdClusterInformerFactory.Start(stopCh)

	select {}
}

func createPod(kubeClient *kubernetes.Clientset, cluster *samplecrdv1.EtcdCluster, index int) {
	state := "new"
	if index == 0 {
		state = "existing"
	}
	podName := rand.String(8)
	commands := fmt.Sprintf("/usr/local/bin/etcd --data-dir=/var/etcd/data --name=%s --initial-advertise-peer-urls=%s "+
		"--listen-peer-urls=%s --listen-client-urls=%s --advertise-client-urls=%s "+
		"--initial-cluster=%s --initial-cluster-state=%s",
		podName, "2380", "0.0.0.0:2380", "0.0.0.0:2379", "2379", strings.Join(nil, ","), state)
	if state == "new" {
		commands = fmt.Sprintf("%s --initial-cluster-token=%s", commands, uuid.New())
	}
	pod := v12.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:        podName,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Spec: v12.PodSpec{
			Containers: []v12.Container{{
				Command: strings.Split(commands, " "),
				Image:   "ss",
				Ports: []v12.ContainerPort{
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
			RestartPolicy: v12.RestartPolicyNever,
		},
	}
	pod.ObjectMeta.Labels["clusterName"] = cluster.Name
	_, err := kubeClient.CoreV1().Pods(cluster.Namespace).Create(context.TODO(), &pod, v1.CreateOptions{})
	if err != nil {
		glog.Fatal(err)
	}
}
