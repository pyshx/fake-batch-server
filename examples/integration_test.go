package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	baseURL  = "http://localhost:8080/v1"
	project  = "test-project"
	location = "us-central1"
)

type Job struct {
	Name       string                 `json:"name"`
	State      string                 `json:"state"`
	Priority   int                    `json:"priority"`
	TaskGroups []TaskGroup            `json:"taskGroups"`
	Labels     map[string]string      `json:"labels"`
	Status     map[string]interface{} `json:"status"`
}

type TaskGroup struct {
	Name      string   `json:"name"`
	TaskSpec  TaskSpec `json:"taskSpec"`
	TaskCount int64    `json:"taskCount"`
}

type TaskSpec struct {
	ComputeResource ComputeResource `json:"computeResource"`
	Runnables       []Runnable      `json:"runnables"`
	MaxRunDuration  string          `json:"maxRunDuration"`
}

type ComputeResource struct {
	CPUMilli  int64 `json:"cpuMilli"`
	MemoryMib int64 `json:"memoryMib"`
}

type Runnable struct {
	Container Container `json:"container"`
}

type Container struct {
	ImageURI string   `json:"imageUri"`
	Commands []string `json:"commands"`
}

func main() {
	fmt.Println("Testing fake-batch-server integration...")
	
	ctx := context.Background()
	
	// Test health check
	if err := testHealthCheck(ctx); err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	
	// Create and monitor a job
	if err := testJobLifecycle(ctx); err != nil {
		log.Fatalf("Job lifecycle test failed: %v", err)
	}
	
	fmt.Println("\nAll tests passed!")
}

func testHealthCheck(ctx context.Context) error {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	
	fmt.Println("✓ Health check passed")
	return nil
}

func testJobLifecycle(ctx context.Context) error {
	// Create job
	job := &Job{
		Priority: 50,
		TaskGroups: []TaskGroup{{
			Name: "task-group-1",
			TaskSpec: TaskSpec{
				ComputeResource: ComputeResource{
					CPUMilli:  2000,
					MemoryMib: 4096,
				},
				Runnables: []Runnable{{
					Container: Container{
						ImageURI: "golang:1.21",
						Commands: []string{"go", "version"},
					},
				}},
				MaxRunDuration: "3600s",
			},
			TaskCount: 5,
		}},
		Labels: map[string]string{
			"test": "integration",
			"lang": "go",
		},
	}
	
	jobName, err := createJob(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to create job: %v", err)
	}
	fmt.Printf("✓ Created job: %s\n", jobName)
	
	// Monitor job progress
	if err := monitorJob(ctx, jobName); err != nil {
		return fmt.Errorf("failed to monitor job: %v", err)
	}
	
	// List jobs
	if err := listJobs(ctx); err != nil {
		return fmt.Errorf("failed to list jobs: %v", err)
	}
	
	// Delete job
	if err := deleteJob(ctx, jobName); err != nil {
		return fmt.Errorf("failed to delete job: %v", err)
	}
	fmt.Printf("✓ Deleted job: %s\n", jobName)
	
	return nil
}

func createJob(ctx context.Context, job *Job) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/jobs?job_id=test-job-%d",
		baseURL, project, location, time.Now().Unix())
	
	resp, err := httpJSON("POST", url, job)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var created Job
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", err
	}
	
	return created.Name, nil
}

func monitorJob(ctx context.Context, jobName string) error {
	url := fmt.Sprintf("%s/%s", baseURL, jobName)
	
	for i := 0; i < 10; i++ {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		
		var job Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()
		
		fmt.Printf("  Job state: %s\n", job.State)
		
		if job.State == "SUCCEEDED" || job.State == "FAILED" {
			fmt.Printf("✓ Job completed with state: %s\n", job.State)
			return nil
		}
		
		time.Sleep(2 * time.Second)
	}
	
	return fmt.Errorf("job did not complete in time")
}

func listJobs(ctx context.Context) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/jobs", baseURL, project, location)
	
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var result struct {
		Jobs []Job `json:"jobs"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	
	fmt.Printf("✓ Listed %d jobs\n", len(result.Jobs))
	return nil
}

func deleteJob(ctx context.Context, jobName string) error {
	url := fmt.Sprintf("%s/%s", baseURL, jobName)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete returned status %d", resp.StatusCode)
	}
	
	return nil
}

func httpJSON(method, url string, body interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

