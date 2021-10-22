package main

import (
	"flag"
	"fmt"
	"github.com/seveirbian/edgeserverless/pkg/backend"
	"github.com/seveirbian/edgeserverless/pkg/entry"
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
	fnAccessor string
)

var (
	stopCh = signals.SetupSignalHandler()

	RulesManager *rulesmanager.RulesManager
	Entry *entry.Entry
	RouteController *controller.RouteController
)

func main() {
	flag.Parse()

	Prepare()

	go Entry.Start()

	err := RouteController.Run(2, stopCh)
	if err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func Prepare() {
	var trace = 1

	// initialize rules manager
	fmt.Printf("[route-proxy] %d initialize rules manager\n", trace)
	trace++
	RulesManager = rulesmanager.NewRulesManager()

	// initialize backends
	fmt.Printf("[route-proxy] %d initialize backends\n", trace)
	trace++
	backend.NewK8sServiceBackend()
	backend.NewYuanrongBackend(fnAccessor)

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

	routeClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	routeInformerFactory := informers.NewSharedInformerFactory(routeClient, time.Second*30)

	RouteController = controller.NewRouteController(kubeClient, routeClient,
		routeInformerFactory.Edgeserverless().V1alpha1().Routes(), RulesManager)

	go routeInformerFactory.Start(stopCh)

	// initialize entry
	fmt.Printf("[route-proxy] %d initialize entry\n", trace)
	trace++
	Entry = entry.NewEntry(RulesManager)
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&fnAccessor, "fnAccessor", "", "The address of FnAccessor. Like 192.168.0.1:11111")
}
