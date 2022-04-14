package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Action 定义的执行动作接口
type Action interface {
	Execute(ctx context.Context) error
}

// PatchStatus 用户更新对象 status 状态
type PatchStatus struct {
	client   client.Client
	original runtime.Object //原有资源对象形式
	new      runtime.Object //最终变成的资源对象形式
}

func (s *PatchStatus) Execute(ctx context.Context) error {
	// 如果对象相等则直接退出
	if reflect.DeepEqual(s.original, s.new) {
		return nil
	}
	// 更新状态
	if err := s.client.Status().Patch(
		ctx,
		s.new,
		client.MergeFrom(s.original),
	); err != nil {
		return fmt.Errorf("修补状态错误时 %q", err)
	}
	return nil
}

// CreateObject 创建一个新的资源对象
type CreateObject struct {
	client client.Client
	obj    runtime.Object
}

func (o *CreateObject) Execute(ctx context.Context) error {
	if err := o.client.Create(ctx, o.obj); err != nil {
		return fmt.Errorf("error %q while creating object ", err)
	}
	return nil
}
