package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobState(t *testing.T) {
	tests := []struct {
		name     string
		state    JobState
		expected string
	}{
		{"Unspecified", JobStateUnspecified, "STATE_UNSPECIFIED"},
		{"Queued", JobStateQueued, "QUEUED"},
		{"Running", JobStateRunning, "RUNNING"},
		{"Succeeded", JobStateSucceeded, "SUCCEEDED"},
		{"Failed", JobStateFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.state))
		})
	}
}

func TestTaskState(t *testing.T) {
	tests := []struct {
		name     string
		state    TaskState
		expected string
	}{
		{"Unspecified", TaskStateUnspecified, "STATE_UNSPECIFIED"},
		{"Pending", TaskStatePending, "PENDING"},
		{"Running", TaskStateRunning, "RUNNING"},
		{"Succeeded", TaskStateSucceeded, "SUCCEEDED"},
		{"Failed", TaskStateFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.state))
		})
	}
}

func TestJobStructure(t *testing.T) {
	job := &Job{
		Name:       "projects/test/locations/us/jobs/job1",
		UID:        "test-uid",
		Priority:   50,
		State:      JobStateQueued,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
		Labels: map[string]string{
			"env":  "test",
			"team": "engineering",
		},
		TaskGroups: []*TaskGroup{
			{
				Name: "group1",
				TaskSpec: &TaskSpec{
					ComputeResource: &ComputeResource{
						CPUMilli:  2000,
						MemoryMib: 4096,
					},
					Runnables: []*Runnable{
						{
							Container: &Container{
								ImageURI: "test:latest",
								Commands: []string{"echo", "test"},
							},
						},
					},
					MaxRunDuration: "3600s",
					MaxRetryCount:  3,
				},
				TaskCount:   10,
				Parallelism: 5,
			},
		},
		AllocationPolicy: &AllocationPolicy{
			Location: &LocationPolicy{
				AllowedLocations: []string{"us-central1-a"},
			},
			ServiceAccount: &ServiceAccount{
				Email: "test@test.iam.gserviceaccount.com",
			},
		},
		LogsPolicy: &LogsPolicy{
			Destination: "CLOUD_LOGGING",
		},
		Status: &JobStatus{
			State: JobStateQueued,
			StatusEvents: []*StatusEvent{
				{
					Type:        "job_created",
					Description: "Job created",
					EventTime:   time.Now(),
				},
			},
		},
	}

	assert.NotNil(t, job)
	assert.Equal(t, "projects/test/locations/us/jobs/job1", job.Name)
	assert.Equal(t, int32(50), job.Priority)
	assert.Len(t, job.TaskGroups, 1)
	assert.Equal(t, "group1", job.TaskGroups[0].Name)
	assert.Equal(t, int64(10), job.TaskGroups[0].TaskCount)
	assert.Equal(t, int64(2000), job.TaskGroups[0].TaskSpec.ComputeResource.CPUMilli)
	assert.Len(t, job.Labels, 2)
	assert.Equal(t, "test", job.Labels["env"])
}

func TestTaskStructure(t *testing.T) {
	task := &Task{
		Name: "projects/test/locations/us/jobs/job1/tasks/task1",
		Status: &TaskStatus{
			State: TaskStatePending,
			StatusEvents: []*StatusEvent{
				{
					Type:        "task_created",
					Description: "Task created",
					EventTime:   time.Now(),
				},
			},
		},
	}

	assert.NotNil(t, task)
	assert.Equal(t, "projects/test/locations/us/jobs/job1/tasks/task1", task.Name)
	assert.NotNil(t, task.Status)
	assert.Equal(t, TaskStatePending, task.Status.State)
	assert.Len(t, task.Status.StatusEvents, 1)
}

func TestEnvironmentVariables(t *testing.T) {
	env := &Environment{
		Variables: map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		},
		SecretVariables: map[string]string{
			"SECRET1": "secret_value1",
		},
	}

	assert.Len(t, env.Variables, 2)
	assert.Equal(t, "value1", env.Variables["KEY1"])
	assert.Len(t, env.SecretVariables, 1)
	assert.Equal(t, "secret_value1", env.SecretVariables["SECRET1"])
}
