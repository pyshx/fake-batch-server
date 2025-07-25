# Fake Batch Server

A standalone emulator for Google Cloud Batch API, similar to fake-gcs-server. This server provides a lightweight, in-memory implementation of the Batch API for local development and testing without requiring any Google Cloud credentials.

## Features

- Full Google Cloud Batch API compatibility
- Create and manage batch jobs
- Submit and monitor tasks
- Automatic job state transitions (QUEUED → RUNNING → SUCCEEDED/FAILED)
- RESTful API compatible with Google Cloud Batch client libraries
- In-memory storage (no persistence)
- Zero configuration required
- Docker support for easy testing

## Quick Start

### Using Docker (Recommended)

```bash
docker run -d -p 8080:8080 fake-batch-server
```

### Using Docker Compose

```bash
docker-compose up -d
```

### Environment Variables

- `PORT` - Server port (default: 8080)
- `HOST` - Server host (default: 0.0.0.0)
- `VERBOSE` - Enable verbose logging (default: false)

## Usage with Google Cloud Client Libraries

Configure your application to use the fake server by setting the endpoint:

### Python
```python
from google.cloud import batch_v1
import google.auth.credentials

# Create fake credentials
class FakeCredentials(google.auth.credentials.Credentials):
    def refresh(self, request):
        pass
    
    @property
    def expired(self):
        return False
    
    @property
    def valid(self):
        return True

# Configure client to use fake server
client_options = {"api_endpoint": "localhost:8080"}
client = batch_v1.BatchServiceClient(
    credentials=FakeCredentials(),
    client_options=client_options
)

# Use the client normally
job = batch_v1.Job()
job.task_groups = [batch_v1.TaskGroup()]
# ... configure job ...

parent = "projects/test-project/locations/us-central1"
response = client.create_job(parent=parent, job=job)
```

### Go
```go
import (
    batch "cloud.google.com/go/batch/apiv1"
    "google.golang.org/api/option"
)

ctx := context.Background()
client, err := batch.NewClient(ctx,
    option.WithEndpoint("localhost:8080"),
    option.WithoutAuthentication(),
    option.WithGRPCDialOption(grpc.WithInsecure()),
)
```

### Java
```java
BatchServiceSettings settings = BatchServiceSettings.newBuilder()
    .setEndpoint("localhost:8080")
    .setCredentialsProvider(NoCredentialsProvider.create())
    .build();

try (BatchServiceClient client = BatchServiceClient.create(settings)) {
    // Use client
}
```

## API Endpoints

- `POST /v1/projects/{project}/locations/{location}/jobs` - Create a job
- `GET /v1/projects/{project}/locations/{location}/jobs` - List jobs
- `GET /v1/projects/{project}/locations/{location}/jobs/{job}` - Get job details
- `DELETE /v1/projects/{project}/locations/{location}/jobs/{job}` - Delete a job
- `GET /v1/projects/{project}/locations/{location}/jobs/{job}/tasks` - List tasks
- `GET /v1/projects/{project}/locations/{location}/jobs/{job}/tasks/{task}` - Get task details
- `GET /v1/health` - Health check endpoint

## Testing

The server automatically simulates job execution:
1. Jobs start in QUEUED state
2. After 2 seconds, transition to RUNNING
3. After 5 more seconds, transition to SUCCEEDED
4. All tasks follow the same pattern

## Building from Source

```bash
# Clone the repository
git clone https://github.com/pyshx/fake-batch-server
cd fake-batch-server

# Build
make build

# Run tests
make test

# Build Docker image
make docker-build
```

## License

MIT
