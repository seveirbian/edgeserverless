package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	edgeserverless "github.com/seveirbian/edgeserverless/pkg/apis/edgeserverless/v1alpha1"
	clientset "github.com/seveirbian/edgeserverless/pkg/client/clientset/versioned"
	routescheme "github.com/seveirbian/edgeserverless/pkg/client/clientset/versioned/scheme"
	informers "github.com/seveirbian/edgeserverless/pkg/client/informers/externalversions/edgeserverless/v1alpha1"
	listers "github.com/seveirbian/edgeserverless/pkg/client/listers/edgeserverless/v1alpha1"
	"github.com/seveirbian/edgeserverless/pkg/rulesmanager"
)

const controllerAgentName = "route-controller"

const (
	SuccessSynced = "Synced"

	MessageResourceSynced = "Route synced successfully"
)

type RouteController struct {
	kubeClientSet kubernetes.Interface

	routeClientSet clientset.Interface

	routesLister listers.RouteLister

	routesSynced cache.InformerSynced

	workQueue workqueue.RateLimitingInterface

	recorder record.EventRecorder

	routeToURI   sync.Map
	rulesManager *rulesmanager.RulesManager
}

// NewRouteController returns a new route controller
func NewRouteController(
	kubeClientSet kubernetes.Interface,
	routeClientSet clientset.Interface,
	routeInformer informers.RouteInformer,
	rulesManager *rulesmanager.RulesManager) *RouteController {

	utilruntime.Must(routescheme.AddToScheme(scheme.Scheme))
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &RouteController{
		kubeClientSet:  kubeClientSet,
		routeClientSet: routeClientSet,
		routesLister:   routeInformer.Lister(),
		routesSynced:   routeInformer.Informer().HasSynced,
		workQueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Routes"),
		recorder:       recorder,
		routeToURI:     sync.Map{},
		rulesManager:   rulesManager,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Route resources change
	routeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueRoute,
		UpdateFunc: func(old, new interface{}) {
			oldRoute := old.(*edgeserverless.Route)
			newRoute := new.(*edgeserverless.Route)
			if oldRoute.ResourceVersion == newRoute.ResourceVersion {
				//??????????????????????????????????????????????????????????????????
				return
			}
			controller.enqueueRoute(new)
		},
		DeleteFunc: controller.enqueueRouteForDelete,
	})

	return controller
}

//???????????????controller?????????
func (c *RouteController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workQueue.ShutDown()

	glog.Info("??????controller???????????????????????????????????????")
	if ok := cache.WaitForCacheSync(stopCh, c.routesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("worker??????")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("worker????????????")
	<-stopCh
	glog.Info("worker????????????")

	return nil
}

func (c *RouteController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// ???????????????
func (c *RouteController) processNextWorkItem() bool {

	obj, shutdown := c.workQueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workQueue.Done.
	err := func(obj interface{}) error {
		defer c.workQueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workQueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// ???syncHandler???????????????
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.workQueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// ??????
func (c *RouteController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// ?????????????????????
	route, err := c.routesLister.Routes(namespace).Get(name)
	if err != nil {
		// ??????Route???????????????????????????????????????????????????????????????????????????
		if errors.IsNotFound(err) {
			glog.Infof("Route?????????????????????????????????????????????????????????: %s/%s ...", namespace, name)
			value, ok := c.routeToURI.Load(key)
			if !ok {
				fmt.Printf("[route controller] no route %s to uri\n", key)
				return nil
			}

			uri, ok := value.(string)
			if !ok {
				return fmt.Errorf("[route controller] uri not string\n")
			}

			c.rulesManager.DeleteRule(uri)
			c.routeToURI.Delete(key)

			return nil
		}

		runtime.HandleError(fmt.Errorf("failed to list route by: %s/%s", namespace, name))

		return err
	}

	glog.Infof("?????????route?????????????????????: %#v ...", route)
	glog.Infof("?????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????????(??????????????????)")

	if value, ok := c.routeToURI.Load(key); ok {
		uri, ok := value.(string)
		if !ok {
			fmt.Printf(fmt.Sprintf("[controller] delete error: not a valid uri string %v\n", value))
			return fmt.Errorf("[controller] delete error: not a valid uri string %v\n", value)
		}
		c.rulesManager.DeleteRule(uri)
		c.routeToURI.Delete(key)
	}
	c.rulesManager.AddRule(route.Spec.URI, route.Spec)
	c.routeToURI.Store(key, route.Spec.URI)

	c.recorder.Event(route, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// ????????????????????????????????????
func (c *RouteController) enqueueRoute(obj interface{}) {
	var key string
	var err error
	// ?????????????????????
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	// ???key????????????
	c.workQueue.AddRateLimited(key)
}

// ????????????
func (c *RouteController) enqueueRouteForDelete(obj interface{}) {
	var key string
	var err error
	// ??????????????????????????????
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	//??????key????????????
	c.workQueue.AddRateLimited(key)
}
