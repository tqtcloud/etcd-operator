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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1beta1 "github.com/tqtcloud/etcd-operator/api/v1beta1"
)

// backupState 包含 EtcdBackup 真实和期望的状态（这里的状态并不是说status）
type backupState struct {
	backup  *etcdv1beta1.EtcdBackup // EtcdBackup 对象本身
	actual  *backupStateContainer   // 真实的状态
	desired *backupStateContainer   // 期望的状态
}

// backupStateContainer 包含 EtcdBackup 的状态
type backupStateContainer struct {
	pod *corev1.Pod
}

// EtcdBackupReconciler reconciles a EtcdBackup object
type EtcdBackupReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	BackupImage string
}

// setStateActual 用于设置 backupState 的真实状态
func (r *EtcdBackupReconciler) setStateActual(ctx context.Context, state *backupState) error {
	var actual backupStateContainer //定义实际状态对象
	key := client.ObjectKey{        //获取对象key   namespace：name
		Name:      state.backup.Name,
		Namespace: state.backup.Namespace,
	}
	// 获取对应的 Pod
	actual.pod = &corev1.Pod{}
	if err := r.Get(ctx, key, actual.pod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("获取Pod出错：%s", err)
		}
		actual.pod = nil
	}

	// 填充当前真实的状态
	state.actual = &actual
	return nil
}

// setStateDesired 用于设置 backupState 的期望状态（根据 EtcdBackup 对象）
func (r EtcdBackupReconciler) setStateDesired(state *backupState) error {
	var desired backupStateContainer //定义期望状态对象

	// 创建一个管理的 Pod 用于执行备份操作
	pod, err := podForBackup(state.backup, r.BackupImage) // r.BackupImage 调用了main中传入的镜像参数，busybox默认
	if err != nil {
		return fmt.Errorf("创建用于备份的Pod失败: %q", err)
	}

	//配置 controller reference 协调 删除了 EtcdBackup 对象的时候对应的 Pod 也需要删除
	if err := controllerutil.SetControllerReference(state.backup, pod, r.Scheme); err != nil {
		return fmt.Errorf("设置 pod 控制器错误 : %s", err)
	}
	desired.pod = pod
	// 获得期望对象
	state.desired = &desired
	return nil
}

// getState 用来获取当前应用的整个状态，然后才方便判断下一步动作
func (r EtcdBackupReconciler) getState(ctx context.Context, req ctrl.Request) (*backupState, error) {
	var state backupState
	// 获取 EtcdBackup 对象
	state.backup = &etcdv1beta1.EtcdBackup{}
	if err := r.Get(ctx, req.NamespacedName, state.backup); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, fmt.Errorf(" 获取 EtcdBackup 对象错误: %s", err)
		}
		// 被删除了则直接忽略
		state.backup = nil
		return &state, nil
	}

	// 获取当前备份的真实状态
	if err := r.setStateActual(ctx, &state); err != nil {
		return nil, fmt.Errorf("设置实际备份状态出错: %s", err)
	}
	// 获取当前期望的状态
	if err := r.setStateDesired(&state); err != nil {
		return nil, fmt.Errorf("设置期望状态错误: %s", err)
	}
	return &state, nil
}

// podForBackup 创建一个 Pod 运行备份任务
func podForBackup(backup *etcdv1beta1.EtcdBackup, image string) (*corev1.Pod, error) {
	// 构造一个全新的备份 Pod
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "backup-agent",
					Image: image, // 执行备份的镜像
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever, //Pod 重启策略
		},
	}, nil
}

//+kubebuilder:rbac:groups=etcd.tqtcloud.io,resources=etcdbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.tqtcloud.io,resources=etcdbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.tqtcloud.io,resources=etcdbackups/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *EtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)

	// get backup state
	state, err := r.getState(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 根据状态来判断下一步要执行的动作
	var action Action

	switch {
	case state.backup == nil: // 被删除了
		logs.Info("Backup Object not found. Ignoring.")
	case !state.backup.DeletionTimestamp.IsZero(): // 标记为了删除
		logs.Info("Backup Object has been deleted. Ignoring.")
	case state.backup.Status.Phase == "": // 开始备份，更新状态
		logs.Info("Backup Staring. Updating status.")
		newBackup := state.backup.DeepCopy()                                            // 深拷贝一份
		newBackup.Status.Phase = etcdv1beta1.EtcdBackupPhaseBackingUp                   // 更新状态为备份中
		action = &PatchStatus{client: r.Client, original: state.backup, new: newBackup} // 下一步要执行的动作
	case state.backup.Status.Phase == etcdv1beta1.EtcdBackupPhaseFailed: // 备份失败
		logs.Info("Backup has failed. Ignoring.")
	case state.backup.Status.Phase == etcdv1beta1.EtcdBackupPhaseCompleted: // 备份完成
		logs.Info("Backup has completed. Ignoring.")
	case state.actual.pod == nil: // 当前还没有备份的 Pod
		logs.Info("Backup Pod does not exists. Creating.")
		action = &CreateObject{client: r.Client, obj: state.desired.pod} // 下一步要执行的动作
	case state.actual.pod.Status.Phase == corev1.PodFailed: // 备份Pod执行失败
		logs.Info("Backup Pod failed. Updating status.")
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1beta1.EtcdBackupPhaseFailed
		action = &PatchStatus{client: r.Client, original: state.backup, new: newBackup} // 下一步更新状态为失败
	case state.actual.pod.Status.Phase == corev1.PodSucceeded: // 备份Pod执行完成
		logs.Info("Backup Pod succeeded. Updating status.")
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1beta1.EtcdBackupPhaseCompleted
		action = &PatchStatus{client: r.Client, original: state.backup, new: newBackup} // 下一步更新状态为完成
	}

	// 执行动作
	if action != nil {
		if err := action.Execute(ctx); err != nil {
			return ctrl.Result{}, fmt.Errorf("executing action error: %s", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1beta1.EtcdBackup{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
