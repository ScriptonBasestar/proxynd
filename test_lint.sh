#!/bin/bash

# Run golangci-lint and capture output
echo "Running golangci-lint..."
golangci-lint run ./handlers/proxy/... 2>&1 | tee lint_output.txt

# Check exit code
LINT_EXIT_CODE=$?
echo "Lint exit code: $LINT_EXIT_CODE"

# Show any typecheck errors
echo -e "\n=== Typecheck errors ==="
grep -i "typecheck" lint_output.txt || echo "No typecheck errors found"

# Show any errors related to WrapAPTError or WrapMavenError
echo -e "\n=== Wrap*Error related issues ==="
grep -E "Wrap(APT|Maven)Error" lint_output.txt || echo "No Wrap*Error issues found"

# Show any undefined errors
echo -e "\n=== Undefined errors ==="
grep -i "undefined" lint_output.txt || echo "No undefined errors found"

exit $LINT_EXIT_CODE
