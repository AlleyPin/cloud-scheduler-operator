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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	scheduler "cloud.google.com/go/scheduler/apiv1"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	schedulerpb "google.golang.org/genproto/googleapis/cloud/scheduler/v1"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcpcontribv1beta1 "github.com/AlleyPin/cloud-scheduler-operator/api/v1beta1"
)

// SchedulerReconciler reconciles a Scheduler object
type SchedulerReconciler struct {
	client.Client
	Log      zerolog.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	CloudScheduler *scheduler.CloudSchedulerClient
}

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=gcp-contrib.alleypinapis.com,resources=schedulers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gcp-contrib.alleypinapis.com,resources=schedulers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gcp-contrib.alleypinapis.com,resources=schedulers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Scheduler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *SchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	r.Log = zlog.With().Interface("cloudschedulerentry", req.NamespacedName).Caller().Logger()

	entry := &gcpcontribv1beta1.Scheduler{}
	err := r.Get(ctx, req.NamespacedName, entry)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info().Msg("CloudScheduler resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		r.Log.Error().Err(err).Msg("Failed to get CloudScheduler")
		return ctrl.Result{}, err
	}

	if entry.Spec.SecretRef == nil {
		return ctrl.Result{}, fmt.Errorf("spec.secretRef is empty, but is required")
	}

	secretFound := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: entry.Spec.SecretRef.Name, Namespace: entry.Spec.SecretRef.Namespace}, secretFound); err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error().Err(err).Msg("secret not found")
			return ctrl.Result{}, err
		}

		r.Log.Error().Err(err).Msg("failed to get secret")
		return ctrl.Result{}, err
	}

	if _, ok := secretFound.Data[entry.Spec.SecretRef.DataKey]; !ok {
		r.Log.Error().Err(err).Msg(fmt.Sprintf("no data for key %s", entry.Spec.SecretRef.DataKey))
		return ctrl.Result{}, fmt.Errorf("failed to get gcp service account")
	}

	googleCloudSchedulerClient, err := scheduler.NewCloudSchedulerClient(ctx, option.WithCredentialsJSON(secretFound.Data[entry.Spec.SecretRef.DataKey]))
	if err != nil {
		r.Log.Error().Err(err).Msg("failed to init a google cloud scheduler api client")
		return ctrl.Result{}, err
	}

	defer googleCloudSchedulerClient.Close()

	if entry.Spec.ProjectID == "" {
		return ctrl.Result{}, fmt.Errorf("spec.projectID is empty, but is required")
	}

	if entry.Spec.LocationID == "" {
		return ctrl.Result{}, fmt.Errorf("spec.locationID is empty, but is required")
	}

	if entry.Spec.Name == "" {
		return ctrl.Result{}, fmt.Errorf("spec.name is empty, but is required")
	}

	option := new(JobRetryConfig)
	if entry.Spec.RetryConfig != nil {
		option.MaxRetryAttempts = entry.Spec.RetryConfig.MaxRetryAttempts
		option.MaxRetryDuration = entry.Spec.RetryConfig.MaxRetryDuration
		option.MinBackoffDuration = entry.Spec.RetryConfig.MinBackoffDuration
		option.MaxBackoffDuration = entry.Spec.RetryConfig.MaxBackoffDuration
		option.MaxDoublings = entry.Spec.RetryConfig.MaxDoublings
	}

	var request *JobRequest
	handler := NewHandler(googleCloudSchedulerClient, entry.Spec.ProjectID, entry.Spec.LocationID, entry.Spec.Name, entry.Spec.AppEngineLocationID)
	if _, ok := handler.GetJob(ctx); !ok { // create job request
		request = handler.CreateJobRequest(ctx, entry.Spec.Schedule, entry.Spec.TimeZone, entry.Spec.Description, WithRetryConfig(option))
	} else {
		request = handler.UpdateJobRequest(ctx, entry.Spec.Schedule, entry.Spec.TimeZone, entry.Spec.Description, WithRetryConfig(option))
	}

	switch entry.Spec.TargetType {
	case gcpcontribv1beta1.PubsubTargetType:
		if entry.Spec.PubsubTarget == nil {
			return ctrl.Result{}, fmt.Errorf("spec.targetType is Pubsub, but pubsubTarget is undefined")
		}

		request.NewPubSubTarget(entry.Spec.PubsubTarget.TopicName, entry.Spec.PubsubTarget.Data, entry.Spec.PubsubTarget.Attributes)
	case gcpcontribv1beta1.AppEngineHttpTargetType:
		if entry.Spec.AppEngineHttpTarget == nil {
			return ctrl.Result{}, fmt.Errorf("spec.targetType is AppEngineHttp, but AppEngineHttpTarget is undefined")
		}

		request = request.NewAppEngineHttpTarget(
			entry.Spec.AppEngineHttpTarget.Method,
			entry.Spec.AppEngineHttpTarget.Service,
			entry.Spec.AppEngineHttpTarget.Version,
			entry.Spec.AppEngineHttpTarget.Instance,
			entry.Spec.AppEngineHttpTarget.Host,
			entry.Spec.AppEngineHttpTarget.RelativeUri,
			entry.Spec.AppEngineHttpTarget.Headers,
			entry.Spec.AppEngineHttpTarget.Body,
		)
	case gcpcontribv1beta1.HttpTargetType:
		if entry.Spec.HttpTarget == nil {
			return ctrl.Result{}, fmt.Errorf("spec.targetType is Http, but HttpTarget is undefined")
		}

		request = request.NewHttpTarget(entry.Spec.HttpTarget.Uri, entry.Spec.HttpTarget.Method, entry.Spec.HttpTarget.Headers, entry.Spec.HttpTarget.Body)
	default:
		return ctrl.Result{}, fmt.Errorf("invalid target type specified")
	}

	job, err := request.Do(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	entry.Status.Name = job.Name
	entry.Status.Description = job.Description
	entry.Status.Schedule = job.Schedule
	entry.Status.TimeZone = job.TimeZone
	entry.Status.UserUpdateTime = metav1.Unix(job.GetUserUpdateTime().GetSeconds(), 0)
	entry.Status.State = job.GetState().String()
	entry.Status.RetryConfig.MaxRetryDuration = job.GetRetryConfig().GetMaxRetryDuration().String()
	entry.Status.RetryConfig.MinBackoffDuration = job.GetRetryConfig().GetMinBackoffDuration().String()
	entry.Status.RetryConfig.MaxBackoffDuration = job.GetRetryConfig().GetMaxBackoffDuration().String()
	entry.Status.RetryConfig.MaxDoublings = int(job.GetRetryConfig().GetMaxDoublings())

	if err := r.Status().Update(ctx, entry); err != nil {
		r.Log.Error().Err(err).Msg("Failed to update CloudScheduler status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpcontribv1beta1.Scheduler{}).
		Complete(r)
}

type handler struct {
	*scheduler.CloudSchedulerClient
	projectID           string
	locationID          string
	name                string
	appEngineLocationID string
}

func NewHandler(c *scheduler.CloudSchedulerClient, p, l, n, app string) *handler {
	return &handler{
		CloudSchedulerClient: c,
		projectID:            p,
		locationID:           l,
		name:                 n,
		appEngineLocationID:  app,
	}
}

func (h *handler) GetJob(ctx context.Context) (*schedulerpb.Job, bool) {
	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", h.projectID, h.locationID, h.name)
	job, err := h.CloudSchedulerClient.GetJob(ctx, &schedulerpb.GetJobRequest{Name: jobName})
	if err != nil {
		if errors.IsNotFound(err) {
			return new(schedulerpb.Job), false
		}

		return new(schedulerpb.Job), false
	}

	return job, true
}

func (h *handler) CreateJobRequest(ctx context.Context, spec string, tz string, description string, options ...JobOption) *JobRequest {
	jb := &JobRequest{
		handler:     h,
		Schedule:    spec,
		TimeZone:    tz,
		Description: description,
	}

	for _, option := range options {
		option(jb)
	}

	jb.CreateJobRequest = &schedulerpb.CreateJobRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", h.projectID, h.appEngineLocationID),
		Job:    nil,
	}

	return jb
}

func (h *handler) UpdateJobRequest(ctx context.Context, spec string, tz string, description string, options ...JobOption) *JobRequest {
	jb := &JobRequest{
		handler:     h,
		Schedule:    spec,
		TimeZone:    tz,
		Description: description,
	}

	for _, option := range options {
		option(jb)
	}

	jb.UpdateJobRequest = &schedulerpb.UpdateJobRequest{
		Job: nil,
	}

	return jb
}

type JobRequest struct {
	*handler

	Name        string
	Schedule    string
	TimeZone    string
	Description string

	RetryConfig *JobRetryConfig

	CreateJobRequest *schedulerpb.CreateJobRequest
	UpdateJobRequest *schedulerpb.UpdateJobRequest
}

// If a job does not complete successfully, it is retried, with exponential backoff, according to the settings in retry config.
// Learn more https://cloud.google.com/scheduler/docs/reference/rpc/google.cloud.scheduler.v1?authuser=1&_ga=2.60975276.-1449617874.1626837834#retryconfig
type JobRetryConfig struct {
	// Maximum number of retry attempts for a failed job
	MaxRetryAttempts int

	// Time limit for retrying a failed job, 0s means unlimited
	MaxRetryDuration string

	// Minimum time to wait before retrying a job after it fails
	MinBackoffDuration string

	// Maximum time to wait before retrying a job after it fails
	MaxBackoffDuration string

	// The time between retries will double max doublings times
	MaxDoublings int
}

type JobOption func(*JobRequest)

func WithRetryConfig(conf *JobRetryConfig) JobOption {
	return func(j *JobRequest) {
		j.RetryConfig = conf
	}
}

func (r *JobRequest) NewPubSubTarget(topic string, data string, attrs map[string]string) *JobRequest {
	maxRetryDuration, _ := time.ParseDuration(r.RetryConfig.MaxRetryDuration)
	minBackoffDuration, _ := time.ParseDuration(r.RetryConfig.MinBackoffDuration)
	maxBackoffDuration, _ := time.ParseDuration(r.RetryConfig.MaxBackoffDuration)

	job := &schedulerpb.Job{
		Name:        fmt.Sprintf("projects/%s/locations/%s/jobs/%s", r.projectID, r.appEngineLocationID, r.handler.name),
		Description: r.Description,
		Target: &schedulerpb.Job_PubsubTarget{
			PubsubTarget: &schedulerpb.PubsubTarget{
				TopicName:  topic,
				Data:       []byte(data),
				Attributes: attrs,
			},
		},
		Schedule: r.Schedule,
		TimeZone: r.TimeZone,
		RetryConfig: &schedulerpb.RetryConfig{
			RetryCount:         int32(r.RetryConfig.MaxRetryAttempts),
			MaxRetryDuration:   durationpb.New(maxRetryDuration),
			MinBackoffDuration: durationpb.New(minBackoffDuration),
			MaxBackoffDuration: durationpb.New(maxBackoffDuration),
			MaxDoublings:       int32(r.RetryConfig.MaxDoublings),
		},
	}

	if r.CreateJobRequest != nil {
		r.CreateJobRequest.Job = job
		return r
	}

	if r.UpdateJobRequest != nil {
		r.UpdateJobRequest.Job = job
		return r
	}

	return nil
}

func (r *JobRequest) NewAppEngineHttpTarget(method string, service string, version string, instance string, host string, relativeUri string, headers map[string]string, body string) *JobRequest {
	maxRetryDuration, _ := time.ParseDuration(r.RetryConfig.MaxRetryDuration)
	minBackoffDuration, _ := time.ParseDuration(r.RetryConfig.MinBackoffDuration)
	maxBackoffDuration, _ := time.ParseDuration(r.RetryConfig.MaxBackoffDuration)

	job := &schedulerpb.Job{
		Name:        fmt.Sprintf("projects/%s/locations/%s/jobs/%s", r.projectID, r.appEngineLocationID, r.handler.name),
		Description: r.Description,
		Target: &schedulerpb.Job_AppEngineHttpTarget{
			AppEngineHttpTarget: &schedulerpb.AppEngineHttpTarget{
				// The HTTP method to use for the request. PATCH and OPTIONS are not permitted.
				HttpMethod: schedulerpb.HttpMethod(schedulerpb.HttpMethod_value[strings.ToUpper(method)]),
				AppEngineRouting: &schedulerpb.AppEngineRouting{
					Service:  service,
					Version:  version,
					Instance: instance,
					Host:     host,
				},
				RelativeUri: relativeUri,
				Headers:     headers,
				// HTTP request body. A request body is allowed only if the HTTP method is
				// POST or PUT. It will result in invalid argument error to set a body on a
				// job with an incompatible [HttpMethod][google.cloud.scheduler.v1.HttpMethod].
				Body: []byte(body),
			},
		},
		Schedule: r.Schedule,
		TimeZone: r.TimeZone,
		RetryConfig: &schedulerpb.RetryConfig{
			RetryCount:         int32(r.RetryConfig.MaxRetryAttempts),
			MaxRetryDuration:   durationpb.New(maxRetryDuration),
			MinBackoffDuration: durationpb.New(minBackoffDuration),
			MaxBackoffDuration: durationpb.New(maxBackoffDuration),
			MaxDoublings:       int32(r.RetryConfig.MaxDoublings),
		},
	}

	if r.CreateJobRequest != nil {
		r.CreateJobRequest.Job = job
		return r
	}

	if r.UpdateJobRequest != nil {
		r.UpdateJobRequest.Job = job
		return r
	}

	return nil
}

func (r *JobRequest) NewHttpTarget(uri string, method string, headers map[string]string, body string) *JobRequest {
	maxRetryDuration, _ := time.ParseDuration(r.RetryConfig.MaxRetryDuration)
	minBackoffDuration, _ := time.ParseDuration(r.RetryConfig.MinBackoffDuration)
	maxBackoffDuration, _ := time.ParseDuration(r.RetryConfig.MaxBackoffDuration)

	job := &schedulerpb.Job{
		Name:        fmt.Sprintf("projects/%s/locations/%s/jobs/%s", r.projectID, r.appEngineLocationID, r.handler.name),
		Description: r.Description,
		Target: &schedulerpb.Job_HttpTarget{
			HttpTarget: &schedulerpb.HttpTarget{
				Uri:        uri,
				HttpMethod: schedulerpb.HttpMethod(schedulerpb.HttpMethod_value[strings.ToUpper(method)]),
				Headers:    headers,
				Body:       []byte(body),
			},
		},
		Schedule: r.Schedule,
		TimeZone: r.TimeZone,
		RetryConfig: &schedulerpb.RetryConfig{
			RetryCount:         int32(r.RetryConfig.MaxRetryAttempts),
			MaxRetryDuration:   durationpb.New(maxRetryDuration),
			MinBackoffDuration: durationpb.New(minBackoffDuration),
			MaxBackoffDuration: durationpb.New(maxBackoffDuration),
			MaxDoublings:       int32(r.RetryConfig.MaxDoublings),
		},
	}

	if r.CreateJobRequest != nil {
		r.CreateJobRequest.Job = job
		return r
	}

	if r.UpdateJobRequest != nil {
		r.UpdateJobRequest.Job = job
		return r
	}

	return nil
}

func (r *JobRequest) Do(ctx context.Context) (*schedulerpb.Job, error) {
	if r.CreateJobRequest != nil {
		resp, err := r.handler.CloudSchedulerClient.CreateJob(ctx, r.CreateJobRequest)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	if r.UpdateJobRequest != nil {
		resp, err := r.handler.CloudSchedulerClient.UpdateJob(ctx, r.UpdateJobRequest)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request undefined")
}
