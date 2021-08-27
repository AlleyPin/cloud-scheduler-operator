/*
Copyright 2021.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type SecretRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	DataKey   string `json:"key,omitempty"`
}

type PubsubTarget struct {
	TopicName  string            `json:"topicName"`
	Data       string            `json:"data"`
	Attributes map[string]string `json:"attributes"`
}

type AppEngineHttpTarget struct {
	Method      string            `json:"method"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Instance    string            `json:"instance"`
	Host        string            `json:"host"`
	RelativeUri string            `json:"relativeUri"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
}

type HttpTarget struct {
	Uri     string            `json:"uri"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type RetryConfig struct {
	// Maximum number of retry attempts for a failed job
	MaxRetryAttempts int `json:"maxRetryAttempts"`

	// Time limit for retrying a failed job, 0s means unlimited
	MaxRetryDuration string `json:"maxRetryDuration"`

	// Minimum time to wait before retrying a job after it fails
	MinBackoffDuration string `json:"minBackoffDuration"`

	// Maximum time to wait before retrying a job after it fails
	MaxBackoffDuration string `json:"maxBackoffDuration"`

	// The time between retries will double max doublings times
	MaxDoublings int `json:"maxDoublings"`
}
type TargetTypeEnum string

const (
	PubsubTargetType        TargetTypeEnum = "Pubsub"
	AppEngineHttpTargetType TargetTypeEnum = "AppEngineHttp"
	HttpTargetType          TargetTypeEnum = "Http"
)

// SchedulerSpec defines the desired state of Scheduler
type SchedulerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// The secretRef is a reference that provides the service account json to access google cloud api
	SecretRef *SecretRef `json:"secretRef"`

	// +kubebuilder:validation:Required
	ProjectID string `json:"projectID"`

	// +kubebuilder:validation:Required
	LocationID string `json:"locationID"`

	// +kubebuilder:validation:Required
	AppEngineLocationID string `json:"appEngineLocationID"`

	// +kubebuilder:validation:Required
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Pubsub;AppEngineHttp;Http
	TargetType TargetTypeEnum `json:"targetType"`

	PubsubTarget        *PubsubTarget        `json:"pubsubTarget,omitempty"`
	AppEngineHttpTarget *AppEngineHttpTarget `json:"appEngineHttpTarget,omitempty"`
	HttpTarget          *HttpTarget          `json:"httpTarget,omitempty"`

	// +kubebuilder:validation:Required
	Schedule string `json:"schedule"`

	// +kubebuilder:validation:Required
	TimeZone string `json:"timeZone"`

	RetryConfig *RetryConfig `json:"retryConfig,omitempty"`
}

// SchedulerStatus defines the observed state of Scheduler
type SchedulerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name        string `json:"name"`
	Description string `json:"description"`
	Schedule    string `json:"schedule"`
	TimeZone    string `json:"timeZone"`

	// Output only. The creation time of the job.
	UserUpdateTime metav1.Time `json:"userUpdateTime"`

	// Output only. State of the job.
	State string `json:"state"`

	RetryConfig RetryConfig `json:"retryConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Scheduler is the Schema for the schedulers API
type Scheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulerSpec   `json:"spec,omitempty"`
	Status SchedulerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SchedulerList contains a list of Scheduler
type SchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Scheduler{}, &SchedulerList{})
}
