package app

import (
	appv1alpha1 "app/pkg/apis/app/v1alpha1"
	"app/pkg/resources/deployment"
	"app/pkg/resources/service"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_app")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new App Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileApp{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("app-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource App
	err = c.Watch(&source.Kind{Type: &appv1alpha1.App{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner App
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1alpha1.App{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileApp implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileApp{}

// ReconcileApp reconciles a App object
type ReconcileApp struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a App object and makes changes based on the state read
// and what is in the App.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling App")

	// Fetch the App instance
	instance := &appv1alpha1.App{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp != nil {
		return reconcile.Result{}, err
	}

	// 判断App关联资源是否存在
	//如果不存在，就创建关联资源
	//如果存在，判断是否需要更新，如果需要更新，执行更新操作，如果不需要更新，正常返回


	deploy := &appsv1.Deployment{}
	//如果error不等于nil，并且err是IsNotFound，说明这个deploy不存在，就需要创建它
	err = r.client.Get(context.TODO(), request.NamespacedName, deploy)
	if err != nil {
		if errors.IsNotFound(err) {

			fmt.Println("Deployment不存在，准备创建deployment")
			//创建deployment和service

			//创建deployment
			deploy := deployment.New(instance)
			err := r.client.Create(context.TODO(), deploy)
			if err != nil {
				return reconcile.Result{}, err
			}

			//创建service
			svc := service.New(instance)
			err = r.client.Create(context.TODO(), svc)
			if err != nil {
				return reconcile.Result{}, err
			}

			data, _ := json.Marshal(instance.Spec)
			if instance.Annotations != nil {
				instance.Annotations["spec"] = string(data)
			} else {
				instance.Annotations = map[string]string{"spec": string(data)}
			}

			//更新instance
			err = r.client.Update(context.TODO(), instance)
			if err != nil {
				return reconcile.Result{}, err
			}

			//如果deployment和service都创建成功就return
			return reconcile.Result{}, nil

		} else {
			//说明获取deployment都已经出错了，那么这一次同步就出错了，需要把这个数据扔回给缓存队列当中
			//下一次再重新处理
			//只要是这个err是非nil的，deploy := &appsv1.Deployment{}这条记录就会被重新扔到缓存队列当中
			//等到下一次同步周期到来的时候再去处理
			return reconcile.Result{}, err
		}
	}

	//走到这，意味着if err == nil，说明拿到了deployment，就需要判断它是否需要更新
	//判断是否需要更新，就是去比较新的spec和旧的spec是否一致，
	//如果一致就不需要更新，如果不一致，就需要更新。

	//先定义老的spec
	oldSpec := appv1alpha1.AppSpec{}

	err = json.Unmarshal([]byte(instance.Annotations["spec"]), &oldSpec)
	if err != nil {
		return reconcile.Result{}, err
	}

	//上面已经拿到instance之后，就可以比较oldspec和instance的spec是否一致
	//这里使用的是reflect.DeepEqual这个方法
	//如果不一致，就需要更新
	if ! reflect.DeepEqual(instance.Spec,oldSpec) {
		//更新关联资源
		newDeploy := deployment.New(instance)
		oldDeploy := &appsv1.Deployment{}
		if err := r.client.Get(context.TODO(), request.NamespacedName, oldDeploy); err != nil {
			//不管err是什么错误，只要拿不到old deployment，就返回，等待下次处理
			return reconcile.Result{}, err
		}

		//拿到oldDeploy之后，一定是把newDeploy.spec 赋值给 oldDeploy.spec
		oldDeploy.Spec = newDeploy.Spec

		//这样设置之后，再去更新oldDeploy，这样才能规避k8s中丢失数据一致性
		err = r.client.Update(context.TODO(), oldDeploy)
		if err != nil {
			return reconcile.Result{}, err
		}

		newSvc := service.New(instance)


		oldSvc := &corev1.Service{}
		if err := r.client.Get(context.TODO(), request.NamespacedName, oldSvc); err != nil {
			//不管err是什么错误，只要拿不到old service，就返回，等待下次处理
			return reconcile.Result{}, err
		}

		//把serivce的ClusterIP记录下来
		oldSvcClusterIp := oldSvc.Spec.ClusterIP

		//拿到oldDeploy之后，一定是把newSvc.spec 赋值给 oldSvc.spec
		oldSvc.Spec = newSvc.Spec

		//保持ClusterIP不变，还是使用之前的ClusterIP，否则会出现报错，报clusterIP是不可更改的
		//报错信息："error":"Service \"example-app\" is invalid:
		// spec.clusterIP: Invalid value: \"\": field is immutable"
		oldSvc.Spec.ClusterIP = oldSvcClusterIp

		//这样设置之后，再去更新oldSvc，这样才能规避k8s中丢失数据一致性
		err = r.client.Update(context.TODO(), oldSvc)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil

	/*
	// Define a new Pod object
	pod := newPodForCR(instance)

	// Set App instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
	*/
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *appv1alpha1.App) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
