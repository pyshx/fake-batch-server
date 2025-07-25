package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"

	"github.com/pyshx/fake-batch-server/pkg/api"
	"github.com/pyshx/fake-batch-server/pkg/handlers"
	"github.com/pyshx/fake-batch-server/pkg/storage"
)

func setupBenchmarkServer() *httptest.Server {
	store := storage.NewMemoryStore()
	handler := handlers.NewHandler(store)

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.CreateJob).Methods("POST")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.ListJobs).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.GetJob).Methods("GET")

	return httptest.NewServer(router)
}

func BenchmarkCreateJob(b *testing.B) {
	server := setupBenchmarkServer()
	defer server.Close()

	client := &http.Client{}
	baseURL := server.URL + "/v1"

	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{
			{
				Name: "bench-group",
				TaskSpec: &api.TaskSpec{
					ComputeResource: &api.ComputeResource{
						CPUMilli:  2000,
						MemoryMib: 4096,
					},
					Runnables: []*api.Runnable{
						{
							Container: &api.Container{
								ImageURI: "test:latest",
								Commands: []string{"echo", "test"},
							},
						},
					},
				},
				TaskCount: 10,
			},
		},
	}

	body, _ := json.Marshal(jobRequest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Post(
			fmt.Sprintf("%s/projects/test-project/locations/us-central1/jobs?job_id=bench-job-%d", baseURL, i),
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkGetJob(b *testing.B) {
	server := setupBenchmarkServer()
	defer server.Close()

	client := &http.Client{}
	baseURL := server.URL + "/v1"

	// Create a job first
	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{{Name: "group1", TaskSpec: &api.TaskSpec{}, TaskCount: 1}},
	}
	body, _ := json.Marshal(jobRequest)
	resp, _ := client.Post(
		baseURL+"/projects/test-project/locations/us-central1/jobs?job_id=bench-get-job",
		"application/json",
		bytes.NewBuffer(body),
	)
	resp.Body.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(baseURL + "/projects/test-project/locations/us-central1/jobs/bench-get-job")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkListJobs(b *testing.B) {
	server := setupBenchmarkServer()
	defer server.Close()

	client := &http.Client{}
	baseURL := server.URL + "/v1"

	// Create 100 jobs
	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{{Name: "group1", TaskSpec: &api.TaskSpec{}, TaskCount: 1}},
	}
	body, _ := json.Marshal(jobRequest)

	for i := 0; i < 100; i++ {
		resp, _ := client.Post(
			fmt.Sprintf("%s/projects/test-project/locations/us-central1/jobs?job_id=list-job-%d", baseURL, i),
			"application/json",
			bytes.NewBuffer(body),
		)
		resp.Body.Close()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(baseURL + "/projects/test-project/locations/us-central1/jobs")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	store := storage.NewMemoryStore()
	handler := handlers.NewHandler(store)

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.CreateJob).Methods("POST")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.ListJobs).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.GetJob).Methods("GET")

	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{}
	baseURL := server.URL + "/v1"

	jobRequest := api.Job{
		TaskGroups: []*api.TaskGroup{{Name: "group1", TaskSpec: &api.TaskSpec{}, TaskCount: 5}},
	}
	body, _ := json.Marshal(jobRequest)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Mix of operations
			switch i % 3 {
			case 0: // Create
				resp, err := client.Post(
					fmt.Sprintf("%s/projects/test-project/locations/us-central1/jobs?job_id=concurrent-%d", baseURL, i),
					"application/json",
					bytes.NewBuffer(body),
				)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()

			case 1: // Get
				resp, err := client.Get(fmt.Sprintf("%s/projects/test-project/locations/us-central1/jobs/concurrent-%d", baseURL, i-1))
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()

			case 2: // List
				resp, err := client.Get(baseURL + "/projects/test-project/locations/us-central1/jobs")
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
			i++
		}
	})
}

func BenchmarkMemoryStore(b *testing.B) {
	b.Run("CreateJob", func(b *testing.B) {
		store := storage.NewMemoryStore()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			job := &api.Job{
				Name: fmt.Sprintf("projects/test/locations/us/jobs/job-%d", i),
				TaskGroups: []*api.TaskGroup{
					{Name: "group1", TaskCount: 10},
				},
			}
			if err := store.CreateJob(job); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetJob", func(b *testing.B) {
		store := storage.NewMemoryStore()
		// Create a job
		job := &api.Job{Name: "projects/test/locations/us/jobs/job-1"}
		store.CreateJob(job)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := store.GetJob(job.Name); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListJobs", func(b *testing.B) {
		store := storage.NewMemoryStore()
		// Create 1000 jobs
		for i := 0; i < 1000; i++ {
			job := &api.Job{
				Name: fmt.Sprintf("projects/test/locations/us/jobs/job-%d", i),
			}
			store.CreateJob(job)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := store.ListJobs("test", "us"); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ConcurrentAccess", func(b *testing.B) {
		store := storage.NewMemoryStore()
		var wg sync.WaitGroup
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			wg.Add(3)
			
			// Concurrent creates
			go func(id int) {
				defer wg.Done()
				job := &api.Job{
					Name: fmt.Sprintf("projects/test/locations/us/jobs/concurrent-%d", id),
				}
				store.CreateJob(job)
			}(i)
			
			// Concurrent reads
			go func() {
				defer wg.Done()
				store.ListJobs("test", "us")
			}()
			
			// Concurrent updates
			go func(id int) {
				defer wg.Done()
				if id > 0 {
					job := &api.Job{
						Name:  fmt.Sprintf("projects/test/locations/us/jobs/concurrent-%d", id-1),
						State: api.JobStateRunning,
					}
					store.UpdateJob(job)
				}
			}(i)
		}
		
		wg.Wait()
	})
}

