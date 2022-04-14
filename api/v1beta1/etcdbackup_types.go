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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BackupStorageType string // 存储对象资源类型定义
type EtcdBackupPhase string   //etcd备份状态类型定义

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EtcdBackupSpec defines the desired state of EtcdBackup
type EtcdBackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of EtcdBackup. Edit etcdbackup_types.go to remove/update
	EtcdUrl      string            `json:"etcdUrl"`
	StorageType  BackupStorageType `json:"storageType"`
	BackupSource `json:",inline"`  // inline 结构化的外部{}没了
}

type BackupSource struct {
	S3  *BackupSource `json:"s3,omitempty"`
	OSS *BackupSource `json:"oss,omitempty"`
}
type S3BackupSource struct {
	Path     string `json:"path"`
	S3Secret string `json:"s3Secret"`
	Endpoint string `json:"endpoint,omitempty"`
}
type OSSBackupSource struct {
	Path      string `json:"path"`
	OSSSecret string `json:"ossSecret"`
	Endpoint  string `json:"endpoint,omitempty"`
}

// etcd  备份状态定义   备份中  成功  失败
var (
	EtcdBackupPhaseBackingUp EtcdBackupPhase = "BackingUp"
	EtcdBackupPhaseCompleted EtcdBackupPhase = "Completed"
	EtcdBackupPhaseFailed    EtcdBackupPhase = "Failed"
)

// EtcdBackupStatus defines the observed state of EtcdBackup
type EtcdBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase EtcdBackupPhase `json:"phase,omitempty"` //状态阶段
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"` //启动时间
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"` // 完成时间
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:status

// EtcdBackup is the Schema for the etcdbackups API
type EtcdBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtcdBackupSpec   `json:"spec,omitempty"`
	Status EtcdBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EtcdBackupList contains a list of EtcdBackup
type EtcdBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EtcdBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EtcdBackup{}, &EtcdBackupList{})
}
