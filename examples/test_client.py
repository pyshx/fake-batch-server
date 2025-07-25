#!/usr/bin/env python3
"""
Example test client for fake-batch-server
Demonstrates how to use the Google Cloud Batch client library with the fake server
"""

import time
import requests
import json


def test_with_direct_api():
    """Test using direct HTTP API calls"""
    base_url = "http://localhost:8080/v1"
    project = "test-project"
    location = "us-central1"
    
    # Create a job
    job_data = {
        "priority": 50,
        "taskGroups": [{
            "name": "task-group-1",
            "taskSpec": {
                "computeResource": {
                    "cpuMilli": 2000,
                    "memoryMib": 4096
                },
                "runnables": [{
                    "container": {
                        "imageUri": "busybox",
                        "commands": ["echo", "Hello from fake-batch-server!"]
                    }
                }],
                "maxRunDuration": "3600s"
            },
            "taskCount": 3,
            "parallelism": 2
        }],
        "labels": {
            "env": "test",
            "team": "engineering"
        }
    }
    
    # Create job
    create_url = f"{base_url}/projects/{project}/locations/{location}/jobs?job_id=test-job-001"
    print(f"Creating job at: {create_url}")
    
    response = requests.post(create_url, json=job_data)
    if response.status_code != 200:
        print(f"Error creating job: {response.status_code} - {response.text}")
        return
    
    job = response.json()
    job_name = job["name"]
    print(f"Created job: {job_name}")
    print(f"Job state: {job['state']}")
    
    # Wait and check job status
    print("\nWaiting for job to start running...")
    time.sleep(3)
    
    # Get job status
    get_url = f"{base_url}/{job_name}"
    response = requests.get(get_url)
    job = response.json()
    print(f"Job state after 3s: {job['state']}")
    
    # List tasks
    tasks_url = f"{base_url}/{job_name}/tasks"
    response = requests.get(tasks_url)
    tasks = response.json()
    print(f"\nNumber of tasks: {len(tasks['tasks'])}")
    for task in tasks['tasks'][:2]:  # Show first 2 tasks
        print(f"  Task: {task['name'].split('/')[-1]} - State: {task['status']['state']}")
    
    # Wait for completion
    print("\nWaiting for job to complete...")
    time.sleep(5)
    
    # Final job status
    response = requests.get(get_url)
    job = response.json()
    print(f"\nFinal job state: {job['state']}")
    print(f"Run duration: {job['status'].get('runDuration', 'N/A')}")
    
    # List all jobs
    list_url = f"{base_url}/projects/{project}/locations/{location}/jobs"
    response = requests.get(list_url)
    jobs = response.json()
    print(f"\nTotal jobs in project: {len(jobs['jobs'])}")
    
    # Delete job
    print(f"\nDeleting job: {job_name}")
    response = requests.delete(get_url)
    if response.status_code == 200:
        print("Job deletion initiated")


def test_health_check():
    """Test the health check endpoint"""
    response = requests.get("http://localhost:8080/v1/health")
    if response.status_code == 200:
        print("Health check passed:", response.json())
    else:
        print("Health check failed:", response.status_code)


if __name__ == "__main__":
    print("Testing fake-batch-server...")
    print("Make sure the server is running on localhost:8080\n")
    
    # Test health check
    test_health_check()
    print()
    
    # Test main functionality
    test_with_direct_api()
