package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	controller2 "k8s_customize_controller/controller"
	v1 "k8s_customize_controller/pkg/apis/bolingcavalry/v1"
	clientset "k8s_customize_controller/pkg/client/clientset/versioned"
	informers "k8s_customize_controller/pkg/client/informers/externalversions"
	"k8s_customize_controller/pkg/signals"
	"time"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()

	// 处理信号量
	stopCh := signals.SetupSignalHandler()

	// 处理入参
	kubeconfig = "/Users/tal/.kube/testconfig"
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		//glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		//glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	studentClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		//glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	fmt.Println("start.....")

	studentInformerFactory := informers.NewSharedInformerFactory(studentClient, time.Second*30)
	//得到controller
	controller := controller2.NewController(kubeClient, studentClient,
		studentInformerFactory.Bolingcavalry().V1().Students())

	//启动informer
	go studentInformerFactory.Start(stopCh)

	//controller开始处理消息
	if err = controller.Run(2, stopCh); err != nil {
		//glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	v1.AddToScheme(scheme.Scheme)
}
