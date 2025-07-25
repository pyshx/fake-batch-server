package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/pyshx/fake-batch-server/pkg/api"
)

type MemoryStore struct {
	mu    sync.RWMutex
	jobs  map[string]*api.Job
	tasks map[string]map[string]*api.Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs:  make(map[string]*api.Job),
		tasks: make(map[string]map[string]*api.Task),
	}
}

func (s *MemoryStore) CreateJob(job *api.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.Name]; exists {
		return fmt.Errorf("job %s already exists", job.Name)
	}

	s.jobs[job.Name] = job
	s.tasks[job.Name] = make(map[string]*api.Task)

	for _, taskGroup := range job.TaskGroups {
		for i := int64(0); i < taskGroup.TaskCount; i++ {
			taskName := fmt.Sprintf("%s/taskGroups/%s/tasks/%d", job.Name, taskGroup.Name, i)
			task := &api.Task{
				Name: taskName,
				Status: &api.TaskStatus{
					State: api.TaskStatePending,
					StatusEvents: []*api.StatusEvent{
						{
							Type:        "task_created",
							Description: "Task created",
							EventTime:   time.Now(),
						},
					},
				},
			}
			s.tasks[job.Name][taskName] = task
		}
	}

	return nil
}

func (s *MemoryStore) GetJob(name string) (*api.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[name]
	if !exists {
		return nil, fmt.Errorf("job %s not found", name)
	}

	return job, nil
}

func (s *MemoryStore) ListJobs(project, location string) ([]*api.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jobs []*api.Job
	prefix := fmt.Sprintf("projects/%s/locations/%s/jobs/", project, location)

	for name, job := range s.jobs {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

func (s *MemoryStore) UpdateJob(job *api.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.Name]; !exists {
		return fmt.Errorf("job %s not found", job.Name)
	}

	job.UpdateTime = time.Now()
	s.jobs[job.Name] = job

	return nil
}

func (s *MemoryStore) DeleteJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; !exists {
		return fmt.Errorf("job %s not found", name)
	}

	delete(s.jobs, name)
	delete(s.tasks, name)

	return nil
}

func (s *MemoryStore) GetTask(jobName, taskName string) (*api.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobTasks, exists := s.tasks[jobName]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobName)
	}

	task, exists := jobTasks[taskName]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskName)
	}

	return task, nil
}

func (s *MemoryStore) ListTasks(jobName string) ([]*api.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobTasks, exists := s.tasks[jobName]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobName)
	}

	var tasks []*api.Task
	for _, task := range jobTasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *MemoryStore) UpdateTask(jobName string, task *api.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobTasks, exists := s.tasks[jobName]
	if !exists {
		return fmt.Errorf("job %s not found", jobName)
	}

	if _, exists := jobTasks[task.Name]; !exists {
		return fmt.Errorf("task %s not found", task.Name)
	}

	jobTasks[task.Name] = task

	return nil
}
