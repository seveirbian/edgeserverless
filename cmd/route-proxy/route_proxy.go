package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/seveirbian/edgeserverless/pkg/client/clientset/versioned"
	informers "github.com/seveirbian/edgeserverless/pkg/client/informers/externalversions"
	"github.com/seveirbian/edgeserverless/pkg/controller"
	"github.com/seveirbian/edgeserverless/pkg/rulesmanager"
	"github.com/seveirbian/edgeserverless/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

var (
	stopCh = signals.SetupSignalHandler()

	RManager *rulesmanager.RulesManager
	RController *controller.RouteController
)

func main() {
	flag.Parse()

	Prepare()

	go func() {
		for {
			RManager.Rules.Range(func(key, value interface{}) bool {
				fmt.Printf("%v %v", key, value)
				return true
			})
			time.Sleep(1)
		}
	}()

	err := RController.Run(2, stopCh)
	if err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func Prepare() {
	var trace = 1

	// initialize rules manager
	fmt.Printf("[route-proxy] %d initialize rules manager\n", trace)
	trace++
	RManager = rulesmanager.NewRulesManager()

	// initialize route controller
	fmt.Printf("[route-proxy] %d initialize route controller\n", trace)
	trace++

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	studentClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	routeInformerFactory := informers.NewSharedInformerFactory(studentClient, time.Second*30)

	RController = controller.NewRouteController(kubeClient, studentClient,
		routeInformerFactory.Edgeserverless().V1alpha1().Routes(), RManager)

	go routeInformerFactory.Start(stopCh)
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
