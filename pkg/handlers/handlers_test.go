package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pyshx/fake-batch-server/pkg/api"
	"github.com/pyshx/fake-batch-server/pkg/storage"
)

func setupTestHandler() *Handler {
	store := storage.NewMemoryStore()
	return NewHandler(store)
}

func setupRouter(handler *Handler) *mux.Router {
	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.CreateJob).Methods("POST")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.ListJobs).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.GetJob).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.DeleteJob).Methods("DELETE")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}/tasks", handler.ListTasks).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}/tasks/{task}", handler.GetTask).Methods("GET")
	
	return router
}

func TestCreateJob(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	jobRequest := api.Job{
		Priority: 50,
		TaskGroups: []*api.TaskGroup{
			{
				Name: "task-group-1",
				TaskSpec: &api.TaskSpec{
					ComputeResource: &api.ComputeResource{
						CPUMilli:  2000,
						MemoryMib: 4096,
					},
					Runnables: []*api.Runnable{
						{
							Container: &api.Container{
								ImageURI: "busybox",
								Commands: []string{"echo", "hello"},
							},
						},
					},
				},
				TaskCount: 2,
			},
		},
		Labels: map[string]string{
			"test": "true",
		},
	}

	body, _ := json.Marshal(jobRequest)
	req := httptest.NewRequest("POST", "/v1/projects/test-project/locations/us-central1/jobs?job_id=test-job-123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Job
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "projects/test-project/locations/us-central1/jobs/test-job-123", response.Name)
	assert.Equal(t, api.JobStateQueued, response.State)
	assert.NotEmpty(t, response.UID)
	assert.Equal(t, jobRequest.Labels, response.Labels)
	assert.Len(t, response.TaskGroups, 1)
	assert.Equal(t, int64(2), response.TaskGroups[0].TaskCount)
}

func TestCreateJob_AutoGenerateID(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{
			{Name: "group1", TaskSpec: &api.TaskSpec{}, TaskCount: 1},
		},
	}

	body, _ := json.Marshal(jobRequest)
	req := httptest.NewRequest("POST", "/v1/projects/test-project/locations/us-central1/jobs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Job
	json.NewDecoder(w.Body).Decode(&response)
	assert.Contains(t, response.Name, "projects/test-project/locations/us-central1/jobs/job-")
}

func TestGetJob(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// First create a job
	job := &api.Job{
		Name:  "projects/test-project/locations/us-central1/jobs/test-job-123",
		State: api.JobStateQueued,
		UID:   "test-uid",
	}
	handler.store.CreateJob(job)

	// Get the job
	req := httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/test-job-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Job
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, job.Name, response.Name)
	assert.Equal(t, job.UID, response.UID)
}

func TestGetJob_NotFound(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	req := httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/non-existent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListJobs(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// Create multiple jobs
	jobs := []*api.Job{
		{Name: "projects/test-project/locations/us-central1/jobs/job1"},
		{Name: "projects/test-project/locations/us-central1/jobs/job2"},
		{Name: "projects/other-project/locations/us-central1/jobs/job3"},
	}

	for _, job := range jobs {
		handler.store.CreateJob(job)
	}

	// List jobs for test-project
	req := httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.ListJobsResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Len(t, response.Jobs, 2)
}

func TestDeleteJob(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// Create a job
	job := &api.Job{
		Name:  "projects/test-project/locations/us-central1/jobs/test-job-123",
		State: api.JobStateQueued,
	}
	handler.store.CreateJob(job)

	// Delete the job
	req := httptest.NewRequest("DELETE", "/v1/projects/test-project/locations/us-central1/jobs/test-job-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for deletion to complete
	time.Sleep(3 * time.Second)

	// Verify job is deleted
	_, err := handler.store.GetJob(job.Name)
	assert.Error(t, err)
}

func TestListTasks(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// Create a job with tasks
	job := &api.Job{
		Name: "projects/test-project/locations/us-central1/jobs/test-job-123",
		TaskGroups: []*api.TaskGroup{
			{Name: "group1", TaskCount: 3},
		},
	}
	handler.store.CreateJob(job)

	// List tasks
	req := httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/test-job-123/tasks", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.ListTasksResponse
	json.NewDecoder(w.Body).Decode(&response)
	assert.Len(t, response.Tasks, 3)
}

func TestGetTask(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// Create a job with tasks
	job := &api.Job{
		Name: "projects/test-project/locations/us-central1/jobs/test-job-123",
		TaskGroups: []*api.TaskGroup{
			{Name: "group1", TaskCount: 1},
		},
	}
	handler.store.CreateJob(job)

	// Get the task list to find task name
	tasks, _ := handler.store.ListTasks(job.Name)
	require.Len(t, tasks, 1)

	// Get specific task
	taskID := tasks[0].Name[len(job.Name+"/tasks/"):]
	req := httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/test-job-123/tasks/"+taskID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.Task
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, tasks[0].Name, response.Name)
}

func TestJobStateTransitions(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// Create a job
	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{
			{Name: "group1", TaskSpec: &api.TaskSpec{}, TaskCount: 1},
		},
	}

	body, _ := json.Marshal(jobRequest)
	req := httptest.NewRequest("POST", "/v1/projects/test-project/locations/us-central1/jobs?job_id=transition-test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Check initial state
	req = httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/transition-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var job api.Job
	json.NewDecoder(w.Body).Decode(&job)
	assert.Equal(t, api.JobStateQueued, job.State)

	// Wait for state transition to RUNNING
	time.Sleep(3 * time.Second)

	req = httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/transition-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&job)
	assert.Equal(t, api.JobStateRunning, job.State)

	// Wait for completion
	time.Sleep(6 * time.Second)

	req = httptest.NewRequest("GET", "/v1/projects/test-project/locations/us-central1/jobs/transition-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&job)
	assert.Equal(t, api.JobStateSucceeded, job.State)
	assert.NotEmpty(t, job.Status.RunDuration)
}

func TestInvalidRequest(t *testing.T) {
	handler := setupTestHandler()
	router := setupRouter(handler)

	// Test with invalid JSON
	req := httptest.NewRequest("POST", "/v1/projects/test-project/locations/us-central1/jobs", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
