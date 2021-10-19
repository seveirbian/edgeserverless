package controller

import (
	"fmt"
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
	informers "github.com/seveirbian/edgeserverless/pkg/client/informers/externalversions/edgeserverless/v1alpha1"
	listers "github.com/seveirbian/edgeserverless/pkg/client/listers/edgeserverless/v1alpha1"
	routescheme "github.com/seveirbian/edgeserverless/pkg/client/clientset/versioned/scheme"

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
}

// NewRouteController returns a new route controller
func NewRouteController(
	kubeClientSet kubernetes.Interface,
	routeClientSet clientset.Interface,
	routeInformer informers.RouteInformer) *RouteController {

	utilruntime.Must(routescheme.AddToScheme(scheme.Scheme))
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &RouteController{
		kubeClientSet:    kubeClientSet,
		routeClientSet: routeClientSet,
		routesLister:   routeInformer.Lister(),
		routesSynced:   routeInformer.Informer().HasSynced,
		workQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Routes"),
		recorder:         recorder,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Route resources change
	routeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueRoute,
		UpdateFunc: func(old, new interface{}) {
			oldRoute := old.(*edgeserverless.Route)
			newRoute := new.(*edgeserverless.Route)
			if oldRoute.ResourceVersion == newRoute.ResourceVersion {
				//版本一致，就表示没有实际更新的操作，立即返回
				return
			}
			controller.enqueueRoute(new)
		},
		DeleteFunc: controller.enqueueRouteForDelete,
	})

	return controller
}

//在此处开始controller的业务
func (c *RouteController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workQueue.ShutDown()

	glog.Info("开始controller业务，开始一次缓存数据同步")
	if ok := cache.WaitForCacheSync(stopCh, c.routesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("worker启动")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("worker已经启动")
	<-stopCh
	glog.Info("worker已经结束")

	return nil
}

func (c *RouteController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// 取数据处理
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
		// 在syncHandler中处理业务
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

// 处理
func (c *RouteController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// 从缓存中取对象
	route, err := c.routesLister.Routes(namespace).Get(name)
	if err != nil {
		// 如果Route对象被删除了，就会走到这里，所以应该在这里加入执行
		if errors.IsNotFound(err) {
			glog.Infof("Route对象被删除，请在这里执行实际的删除业务: %s/%s ...", namespace, name)

			return nil
		}

		runtime.HandleError(fmt.Errorf("failed to list route by: %s/%s", namespace, name))

		return err
	}

	glog.Infof("这里是route对象的期望状态: %#v ...", route)
	glog.Infof("实际状态是从业务层面得到的，此处应该去的实际状态，与期望状态做对比，并根据差异做出响应(新增或者删除)")

	c.recorder.Event(route, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// 数据先放入缓存，再入队列
func (c *RouteController) enqueueRoute(obj interface{}) {
	var key string
	var err error
	// 将对象放入缓存
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	// 将key放入队列
	c.workQueue.AddRateLimited(key)
}

// 删除操作
func (c *RouteController) enqueueRouteForDelete(obj interface{}) {
	var key string
	var err error
	// 从缓存中删除指定对象
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	//再将key放入队列
	c.workQueue.AddRateLimited(key)
}