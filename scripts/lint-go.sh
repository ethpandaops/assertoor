#!/bin/bash

# Run gofmt to see if there are any formatting changes to be made
if [ $(gofmt -s -l . | wc -l) -gt 0 ]; then
  echo "Running gofmt to fix the code..."
  gofmt -s -w .
else
  echo "The code is already properly formatted with gofmt."
fi

# Run golangci-lint to fix any issues automatically where applicable
echo "Running golangci-lint to apply automatic fixes..."
golangci-lint run --fix

# Now run staticcheck to analyze the code
echo "Running staticcheck..."
staticcheck ./...
