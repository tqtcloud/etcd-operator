/*
Copyright 2022 tqtcloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1beta1 "github.com/tqtcloud/etcd-operator/api/v1beta1"
)

// EtcdClusterReconciler reconciles a EtcdCluster object
type EtcdClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=etcd.tqtcloud.io,resources=etcdclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.tqtcloud.io,resources=etcdclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.tqtcloud.io,resources=etcdclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *EtcdClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)

	// 首先我们获取 EtcdCluster 实例
	var etcdCluster etcdv1beta1.EtcdCluster
	if err := r.Get(ctx, req.NamespacedName, &etcdCluster); err != nil {
		// 如果 EtcdCluster 是被删除的，那么我们应该忽略掉
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 已经获取到了 EtcdCluster 实例
	// 创建/更新对应的 StatefulSet 以及 Headless SVC 对象
	// CreateOrUpdate
	// 调谐：观察当前的状态和期望的状态进行对比
	// CreateOrUpdate Service
	var svc corev1.Service
	svc.Name = etcdCluster.Name
	svc.Namespace = etcdCluster.Namespace
	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		// 调谐的函数必须在这里面实现，实际上就是去拼装我们的 Service
		MutateHeadlessSvc(&etcdCluster, &svc) //创建无头svc
		return controllerutil.SetControllerReference(&etcdCluster, &svc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	logs.Info("CreateOrUpdate", "Service", or)

	// CreateOrUpdate StatefulSet
	var sts appsv1.StatefulSet
	sts.Name = etcdCluster.Name
	sts.Namespace = etcdCluster.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &sts, func() error {
		// 调谐的函数必须在这里面实现，实际上就是去拼装我们的 StatefulSet
		MutateStatefulSet(&etcdCluster, &sts)
		return ctrl.SetControllerReference(&etcdCluster, &sts, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	logs.Info("CreateOrUpdate", "StatefulSet", or)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1beta1.EtcdCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
