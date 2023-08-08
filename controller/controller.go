package controller

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	bolingcavalryv1 "k8s_customize_controller/pkg/apis/bolingcavalry/v1"
	clientset "k8s_customize_controller/pkg/client/clientset/versioned"
	informers "k8s_customize_controller/pkg/client/informers/externalversions/bolingcavalry/v1"
	listers "k8s_customize_controller/pkg/client/listers/bolingcavalry/v1"
	"time"
)

const controllerAgentName = "student-controller"

const (
	SuccessSynced         = "Synced"
	MessageResourceSynced = "Student synced successfully"
)

// Controller is the controller implementation for Student resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset    kubernetes.Interface
	studentclientset clientset.Interface
	studentLister    listers.StudentLister
	studentSynced    cache.InformerSynced
	workqueue        workqueue.RateLimitingInterface
	recorder         record.EventRecorder
}

// NewController returns a new student controller
func NewController(
	kubeclientset kubernetes.Interface,
	studentclientset clientset.Interface,
	studentInformer informers.StudentInformer) *Controller {
	eventBroadcaster := record.NewBroadcaster()

	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:    kubeclientset,
		studentclientset: studentclientset,
		studentLister:    studentInformer.Lister(),
		studentSynced:    studentInformer.Informer().HasSynced,
		workqueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		recorder:         recorder,
	}

	studentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			stu := obj.(*bolingcavalryv1.Student)
			msg := fmt.Sprintf("Student %s/%s added", stu.Namespace, stu.Name)
			fmt.Println(msg)
			if ownerRef := metav1.GetControllerOf(stu); ownerRef != nil {
				fmt.Println("ownerRef.Kind = ", ownerRef.Kind)
			}
			controller.enqueueStudent(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			oldStudent := old.(*bolingcavalryv1.Student)
			newStudent := new.(*bolingcavalryv1.Student)
			if oldStudent.ResourceVersion == newStudent.ResourceVersion {
				//版本一致，就表示没有实际更新的操作，立即返回
				return
			}
			controller.enqueueStudent(new)
		},
		DeleteFunc: func(obj interface{}) {
			// 删除操作，直接将对象放入队列
			controller.enqueueStudentForDelete(obj)
		},
	})

	return controller
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// 等待缓存同步
	if ok := cache.WaitForCacheSync(stopCh, c.studentSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// 启动worker
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {

	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {

			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// 在syncHandler中处理业务
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// 数据先放入缓存，再入队列
func (c *Controller) enqueueStudent(obj interface{}) {
	var key string
	var err error
	// 将对象放入缓存
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	// 将key放入队列
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// 从缓存中取对象
	student, err := c.studentLister.Students(namespace).Get(name)
	if err != nil {
		// 如果Student对象被删除了，就会走到这里，所以应该在这里加入执行
		if errors.IsNotFound(err) {

			return nil
		}

		runtime.HandleError(fmt.Errorf("failed to list student by: %s/%s", namespace, name))

		return err
	}

	c.recorder.Event(student, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) syncDeleteStudent(student *bolingcavalryv1.Student) error {
	// 删除对应的pod
	return nil
}

func (c *Controller) syncCreateStudent(student *bolingcavalryv1.Student) error {
	// 创建对应的pod
	return nil
}

// 删除操作
func (c *Controller) enqueueStudentForDelete(obj interface{}) {
	var key string
	var err error
	// 从缓存中删除指定对象
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	//再将key放入队列
	c.workqueue.AddRateLimited(key)
}
