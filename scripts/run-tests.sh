#!/bin/bash

set -e

echo "Starting fake-batch-server for testing..."

# Start the server in the background
./fake-batch-server &
SERVER_PID=$!

# Wait for server to be ready
echo "Waiting for server to start..."
for i in {1..10}; do
    if curl -s http://localhost:8080/v1/health > /dev/null; then
        echo "Server is ready!"
        break
    fi
    sleep 1
done

# Run tests
echo -e "\n=== Running Python test client ==="
python3 examples/test_client.py

echo -e "\n=== Running Go integration test ==="
go run examples/integration_test.go

# Stop the server
echo -e "\nStopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo -e "\nAll tests completed successfully!"
