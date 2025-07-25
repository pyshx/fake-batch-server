package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pyshx/fake-batch-server/pkg/api"
	"github.com/pyshx/fake-batch-server/pkg/handlers"
	"github.com/pyshx/fake-batch-server/pkg/storage"
)

func setupTestServer() *httptest.Server {
	store := storage.NewMemoryStore()
	handler := handlers.NewHandler(store)

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.CreateJob).Methods("POST")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.ListJobs).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.GetJob).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.DeleteJob).Methods("DELETE")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}/tasks", handler.ListTasks).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}/tasks/{task}", handler.GetTask).Methods("GET")
	v1.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	return httptest.NewServer(router)
}

func TestEndToEnd_CompleteJobLifecycle(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := server.URL + "/v1"

	// 1. Health check
	resp, err := client.Get(baseURL + "/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// 2. Create a job
	jobRequest := api.Job{
		Priority: 100,
		TaskGroups: []*api.TaskGroup{
			{
				Name: "main-group",
				TaskSpec: &api.TaskSpec{
					ComputeResource: &api.ComputeResource{
						CPUMilli:    4000,
						MemoryMib:   8192,
						BootDiskMib: 10240,
					},
					Runnables: []*api.Runnable{
						{
							Container: &api.Container{
								ImageURI: "gcr.io/test/worker:latest",
								Commands: []string{"/bin/worker", "--task"},
							},
							Environment: &api.Environment{
								Variables: map[string]string{
									"TASK_TYPE": "process",
									"INPUT_URL": "gs://bucket/input.txt",
								},
							},
						},
					},
					MaxRunDuration: "3600s",
					MaxRetryCount:  3,
				},
				TaskCount:   5,
				Parallelism: 2,
			},
		},
		AllocationPolicy: &api.AllocationPolicy{
			Location: &api.LocationPolicy{
				AllowedLocations: []string{"us-central1-a", "us-central1-b"},
			},
			ServiceAccount: &api.ServiceAccount{
				Email: "worker@project.iam.gserviceaccount.com",
			},
		},
		Labels: map[string]string{
			"environment": "test",
			"team":        "engineering",
		},
	}

	body, _ := json.Marshal(jobRequest)
	resp, err = client.Post(
		baseURL+"/projects/test-project/locations/us-central1/jobs?job_id=e2e-test-job",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var createdJob api.Job
	err = json.NewDecoder(resp.Body).Decode(&createdJob)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "projects/test-project/locations/us-central1/jobs/e2e-test-job", createdJob.Name)
	assert.Equal(t, api.JobStateQueued, createdJob.State)
	assert.NotEmpty(t, createdJob.UID)

	// 3. Get job details
	resp, err = client.Get(baseURL + "/" + createdJob.Name)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrievedJob api.Job
	json.NewDecoder(resp.Body).Decode(&retrievedJob)
	resp.Body.Close()
	assert.Equal(t, createdJob.Name, retrievedJob.Name)

	// 4. List tasks
	resp, err = client.Get(baseURL + "/" + createdJob.Name + "/tasks")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var taskList api.ListTasksResponse
	json.NewDecoder(resp.Body).Decode(&taskList)
	resp.Body.Close()
	assert.Len(t, taskList.Tasks, 5)

	// 5. Get individual task
	if len(taskList.Tasks) > 0 {
		taskName := taskList.Tasks[0].Name
		taskID := taskName[len(createdJob.Name+"/tasks/"):]
		resp, err = client.Get(fmt.Sprintf("%s/%s/tasks/%s", baseURL, createdJob.Name, taskID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var task api.Task
		json.NewDecoder(resp.Body).Decode(&task)
		resp.Body.Close()
		assert.Equal(t, taskName, task.Name)
		assert.NotNil(t, task.Status)
	}

	// 6. Monitor job state transitions
	checkJobState := func(expectedState api.JobState) {
		resp, err := client.Get(baseURL + "/" + createdJob.Name)
		require.NoError(t, err)
		var job api.Job
		json.NewDecoder(resp.Body).Decode(&job)
		resp.Body.Close()
		assert.Equal(t, expectedState, job.State)
	}

	// Initial state should be QUEUED
	checkJobState(api.JobStateQueued)

	// Wait and check for RUNNING state
	time.Sleep(3 * time.Second)
	checkJobState(api.JobStateRunning)

	// Wait and check for SUCCEEDED state
	time.Sleep(6 * time.Second)
	checkJobState(api.JobStateSucceeded)

	// 7. List all jobs
	resp, err = client.Get(baseURL + "/projects/test-project/locations/us-central1/jobs")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jobList api.ListJobsResponse
	json.NewDecoder(resp.Body).Decode(&jobList)
	resp.Body.Close()
	assert.GreaterOrEqual(t, len(jobList.Jobs), 1)

	// 8. Delete the job
	req, _ := http.NewRequest("DELETE", baseURL+"/"+createdJob.Name, nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Wait for deletion
	time.Sleep(3 * time.Second)

	// Verify job is deleted
	resp, err = client.Get(baseURL + "/" + createdJob.Name)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestEndToEnd_MultipleJobs(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	baseURL := server.URL + "/v1"

	// Create multiple jobs concurrently
	numJobs := 5
	jobNames := make([]string, numJobs)
	
	for i := 0; i < numJobs; i++ {
		go func(idx int) {
			jobRequest := api.Job{
				Priority: int32(idx * 10),
				TaskGroups: []*api.TaskGroup{
					{
						Name:      fmt.Sprintf("group-%d", idx),
						TaskSpec:  &api.TaskSpec{},
						TaskCount: int64(idx + 1),
					},
				},
			}

			body, _ := json.Marshal(jobRequest)
			resp, err := client.Post(
				fmt.Sprintf("%s/projects/test-project/locations/us-central1/jobs?job_id=multi-job-%d", baseURL, idx),
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var job api.Job
			json.NewDecoder(resp.Body).Decode(&job)
			resp.Body.Close()
			jobNames[idx] = job.Name
		}(i)
	}

	// Wait for all jobs to be created
	time.Sleep(1 * time.Second)

	// List all jobs
	resp, err := client.Get(baseURL + "/projects/test-project/locations/us-central1/jobs")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jobList api.ListJobsResponse
	json.NewDecoder(resp.Body).Decode(&jobList)
	resp.Body.Close()
	assert.GreaterOrEqual(t, len(jobList.Jobs), numJobs)
}

func TestEndToEnd_ErrorCases(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	baseURL := server.URL + "/v1"

	// Test 1: Invalid JSON
	resp, err := client.Post(
		baseURL+"/projects/test-project/locations/us-central1/jobs",
		"application/json",
		bytes.NewBufferString("invalid json"),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Test 2: Get non-existent job
	resp, err = client.Get(baseURL + "/projects/test-project/locations/us-central1/jobs/non-existent")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// Test 3: Create duplicate job
	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{{Name: "group1", TaskSpec: &api.TaskSpec{}, TaskCount: 1}},
	}
	body, _ := json.Marshal(jobRequest)

	// Create first job
	resp, err = client.Post(
		baseURL+"/projects/test-project/locations/us-central1/jobs?job_id=duplicate-test",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Try to create duplicate
	resp, err = client.Post(
		baseURL+"/projects/test-project/locations/us-central1/jobs?job_id=duplicate-test",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()
}

