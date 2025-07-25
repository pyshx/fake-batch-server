package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pyshx/fake-batch-server/pkg/api"
)

func TestMemoryStore_CreateJob(t *testing.T) {
	store := NewMemoryStore()

	job := &api.Job{
		Name:  "projects/test/locations/us-central1/jobs/test-job-1",
		UID:   "test-uid",
		State: api.JobStateQueued,
		TaskGroups: []*api.TaskGroup{
			{
				Name:      "group1",
				TaskCount: 3,
			},
		},
	}

	err := store.CreateJob(job)
	assert.NoError(t, err)

	// Verify job was created
	retrieved, err := store.GetJob(job.Name)
	assert.NoError(t, err)
	assert.Equal(t, job.Name, retrieved.Name)
	assert.Equal(t, job.UID, retrieved.UID)

	// Verify tasks were created
	tasks, err := store.ListTasks(job.Name)
	assert.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Verify duplicate job creation fails
	err = store.CreateJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMemoryStore_GetJob(t *testing.T) {
	store := NewMemoryStore()

	// Test non-existent job
	_, err := store.GetJob("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create and retrieve job
	job := &api.Job{
		Name: "projects/test/locations/us-central1/jobs/test-job-1",
	}
	store.CreateJob(job)

	retrieved, err := store.GetJob(job.Name)
	assert.NoError(t, err)
	assert.Equal(t, job.Name, retrieved.Name)
}

func TestMemoryStore_ListJobs(t *testing.T) {
	store := NewMemoryStore()

	// Create jobs in different projects
	jobs := []*api.Job{
		{Name: "projects/project1/locations/us-central1/jobs/job1"},
		{Name: "projects/project1/locations/us-central1/jobs/job2"},
		{Name: "projects/project2/locations/us-central1/jobs/job3"},
		{Name: "projects/project1/locations/us-west1/jobs/job4"},
	}

	for _, job := range jobs {
		err := store.CreateJob(job)
		require.NoError(t, err)
	}

	// List jobs for project1/us-central1
	listed, err := store.ListJobs("project1", "us-central1")
	assert.NoError(t, err)
	assert.Len(t, listed, 2)

	// List jobs for project2/us-central1
	listed, err = store.ListJobs("project2", "us-central1")
	assert.NoError(t, err)
	assert.Len(t, listed, 1)

	// List jobs for project1/us-west1
	listed, err = store.ListJobs("project1", "us-west1")
	assert.NoError(t, err)
	assert.Len(t, listed, 1)
}

func TestMemoryStore_UpdateJob(t *testing.T) {
	store := NewMemoryStore()

	// Test updating non-existent job
	job := &api.Job{Name: "non-existent"}
	err := store.UpdateJob(job)
	assert.Error(t, err)

	// Create and update job
	job = &api.Job{
		Name:       "projects/test/locations/us-central1/jobs/test-job-1",
		State:      api.JobStateQueued,
		UpdateTime: time.Now().Add(-1 * time.Hour),
	}
	store.CreateJob(job)

	oldUpdateTime := job.UpdateTime
	job.State = api.JobStateRunning
	err = store.UpdateJob(job)
	assert.NoError(t, err)

	retrieved, _ := store.GetJob(job.Name)
	assert.Equal(t, api.JobStateRunning, retrieved.State)
	assert.True(t, retrieved.UpdateTime.After(oldUpdateTime))
}

func TestMemoryStore_DeleteJob(t *testing.T) {
	store := NewMemoryStore()

	// Test deleting non-existent job
	err := store.DeleteJob("non-existent")
	assert.Error(t, err)

	// Create and delete job
	job := &api.Job{
		Name: "projects/test/locations/us-central1/jobs/test-job-1",
		TaskGroups: []*api.TaskGroup{
			{Name: "group1", TaskCount: 2},
		},
	}
	store.CreateJob(job)

	// Verify job and tasks exist
	_, err = store.GetJob(job.Name)
	assert.NoError(t, err)
	tasks, _ := store.ListTasks(job.Name)
	assert.Len(t, tasks, 2)

	// Delete job
	err = store.DeleteJob(job.Name)
	assert.NoError(t, err)

	// Verify job and tasks are deleted
	_, err = store.GetJob(job.Name)
	assert.Error(t, err)
	_, err = store.ListTasks(job.Name)
	assert.Error(t, err)
}

func TestMemoryStore_Tasks(t *testing.T) {
	store := NewMemoryStore()

	jobName := "projects/test/locations/us-central1/jobs/test-job-1"
	job := &api.Job{
		Name: jobName,
		TaskGroups: []*api.TaskGroup{
			{Name: "group1", TaskCount: 2},
		},
	}
	store.CreateJob(job)

	// List tasks
	tasks, err := store.ListTasks(jobName)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Get specific task
	task, err := store.GetTask(jobName, tasks[0].Name)
	assert.NoError(t, err)
	assert.Equal(t, tasks[0].Name, task.Name)
	assert.Equal(t, api.TaskStatePending, task.Status.State)

	// Update task
	task.Status.State = api.TaskStateRunning
	err = store.UpdateTask(jobName, task)
	assert.NoError(t, err)

	// Verify update
	updated, err := store.GetTask(jobName, task.Name)
	assert.NoError(t, err)
	assert.Equal(t, api.TaskStateRunning, updated.Status.State)

	// Test non-existent job
	_, err = store.GetTask("non-existent", "task")
	assert.Error(t, err)

	// Test non-existent task
	_, err = store.GetTask(jobName, "non-existent-task")
	assert.Error(t, err)
}

func TestMemoryStore_Concurrency(t *testing.T) {
	store := NewMemoryStore()

	// Test concurrent job creation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			job := &api.Job{
				Name: fmt.Sprintf("projects/test/locations/us-central1/jobs/job-%d", id),
			}
			err := store.CreateJob(job)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all jobs were created
	jobs, err := store.ListJobs("test", "us-central1")
	assert.NoError(t, err)
	assert.Len(t, jobs, 10)
}
