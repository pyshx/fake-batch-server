package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/pyshx/fake-batch-server/pkg/api"
	"github.com/pyshx/fake-batch-server/pkg/storage"
)

type Handler struct {
	store *storage.MemoryStore
}

func NewHandler(store *storage.MemoryStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]
	location := vars["location"]

	var job api.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		jobID = fmt.Sprintf("job-%s", uuid.New().String()[:8])
	}

	job.Name = fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)
	job.UID = uuid.New().String()
	job.State = api.JobStateQueued
	job.CreateTime = time.Now()
	job.UpdateTime = job.CreateTime

	if job.Status == nil {
		job.Status = &api.JobStatus{
			State: api.JobStateQueued,
			StatusEvents: []*api.StatusEvent{
				{
					Type:        "job_created",
					Description: "Job created",
					EventTime:   job.CreateTime,
				},
			},
			TaskGroups: make(map[string]*api.TaskGroupStatus),
		}
	}

	for _, taskGroup := range job.TaskGroups {
		job.Status.TaskGroups[taskGroup.Name] = &api.TaskGroupStatus{
			Counts: map[string]int64{
				"PENDING": taskGroup.TaskCount,
			},
		}
	}

	if err := h.store.CreateJob(&job); err != nil {
		writeError(w, http.StatusConflict, "Failed to create job: %v", err)
		return
	}

	go h.simulateJobExecution(&job)

	logrus.Infof("Created job: %s", job.Name)
	writeJSON(w, http.StatusOK, &job)
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]
	location := vars["location"]
	jobID := vars["job"]

	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)

	job, err := h.store.GetJob(jobName)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found: %v", err)
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]
	location := vars["location"]

	jobs, err := h.store.ListJobs(project, location)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list jobs: %v", err)
		return
	}

	response := &api.ListJobsResponse{
		Jobs: jobs,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]
	location := vars["location"]
	jobID := vars["job"]

	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)

	job, err := h.store.GetJob(jobName)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found: %v", err)
		return
	}

	job.State = api.JobStateDeleting
	job.UpdateTime = time.Now()
	if err := h.store.UpdateJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update job: %v", err)
		return
	}

	go func() {
		time.Sleep(2 * time.Second)
		if err := h.store.DeleteJob(jobName); err != nil {
			logrus.Errorf("Failed to delete job %s: %v", jobName, err)
		}
	}()

	logrus.Infof("Deleting job: %s", jobName)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]
	location := vars["location"]
	jobID := vars["job"]

	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)

	tasks, err := h.store.ListTasks(jobName)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found: %v", err)
		return
	}

	response := &api.ListTasksResponse{
		Tasks: tasks,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project := vars["project"]
	location := vars["location"]
	jobID := vars["job"]
	taskID := vars["task"]

	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)
	taskName := fmt.Sprintf("%s/tasks/%s", jobName, taskID)

	task, err := h.store.GetTask(jobName, taskName)
	if err != nil {
		writeError(w, http.StatusNotFound, "Task not found: %v", err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) simulateJobExecution(job *api.Job) {
	time.Sleep(2 * time.Second)

	job.State = api.JobStateRunning
	job.UpdateTime = time.Now()
	job.Status.State = api.JobStateRunning
	job.Status.StatusEvents = append(job.Status.StatusEvents, &api.StatusEvent{
		Type:        "job_started",
		Description: "Job started running",
		EventTime:   time.Now(),
	})

	if err := h.store.UpdateJob(job); err != nil {
		logrus.Errorf("Failed to update job state: %v", err)
		return
	}

	tasks, _ := h.store.ListTasks(job.Name)
	for _, task := range tasks {
		task.Status.State = api.TaskStateRunning
		task.Status.StatusEvents = append(task.Status.StatusEvents, &api.StatusEvent{
			Type:        "task_started",
			Description: "Task started running",
			EventTime:   time.Now(),
		})
		h.store.UpdateTask(job.Name, task)
	}

	for _, taskGroup := range job.TaskGroups {
		job.Status.TaskGroups[taskGroup.Name].Counts = map[string]int64{
			"RUNNING": taskGroup.TaskCount,
		}
	}
	h.store.UpdateJob(job)

	time.Sleep(5 * time.Second)

	for _, task := range tasks {
		task.Status.State = api.TaskStateSucceeded
		task.Status.StatusEvents = append(task.Status.StatusEvents, &api.StatusEvent{
			Type:        "task_completed",
			Description: "Task completed successfully",
			EventTime:   time.Now(),
		})
		h.store.UpdateTask(job.Name, task)
	}

	job.State = api.JobStateSucceeded
	job.UpdateTime = time.Now()
	job.Status.State = api.JobStateSucceeded
	job.Status.StatusEvents = append(job.Status.StatusEvents, &api.StatusEvent{
		Type:        "job_completed",
		Description: "Job completed successfully",
		EventTime:   time.Now(),
	})
	job.Status.RunDuration = "7s"

	for _, taskGroup := range job.TaskGroups {
		job.Status.TaskGroups[taskGroup.Name].Counts = map[string]int64{
			"SUCCEEDED": taskGroup.TaskCount,
		}
	}

	if err := h.store.UpdateJob(job); err != nil {
		logrus.Errorf("Failed to update job state: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logrus.Errorf("Failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	logrus.Error(message)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
