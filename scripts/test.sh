#!/bin/bash

echo "Running fake-batch-server tests..."
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test categories
declare -a test_categories=(
    "Unit Tests:./pkg/storage/..."
    "Handler Tests:./pkg/handlers/..."
    "E2E Tests:./test/..."
)

# Run each test category
for category in "${test_categories[@]}"; do
    IFS=':' read -r name path <<< "$category"
    echo "=== Running $name ==="
    
    if go test -v $path; then
        echo -e "${GREEN}✓ $name passed${NC}"
    else
        echo -e "${RED}✗ $name failed${NC}"
        exit 1
    fi
    echo ""
done

echo "=== Running Benchmarks ==="
go test -bench=. -benchmem -run=^$ ./test/...

echo ""
echo -e "${GREEN}All tests completed!${NC}"

