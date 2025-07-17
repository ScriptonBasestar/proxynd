#!/bin/bash

# Script to help identify and suggest fixes for unchecked defer Close() calls

echo "=== Defer Close() Error Check Analysis ==="
echo ""

# Create a temporary file for results
RESULT_FILE="defer_close_fixes.md"

cat > "$RESULT_FILE" << 'EOF'
# Defer Close() Error Handling Fixes

This file contains suggested fixes for unchecked defer Close() calls.

## Files Requiring Fixes

EOF

# Find all defer Close() calls without error checking
echo "Analyzing defer Close() patterns..."

# Function to generate fix suggestion based on context
generate_fix() {
    local file=$1
    local line_num=$2
    local line_content=$3
    
    # Extract the variable name being closed
    var_name=$(echo "$line_content" | sed -n 's/.*defer \([a-zA-Z0-9_]*\)\.Close().*/\1/p')
    
    # Determine the appropriate logger based on file path
    logger="log"
    if [[ "$file" == *"internal/logging"* ]]; then
        logger="// Use appropriate logger"
    elif [[ "$file" == *"test"* ]]; then
        logger="t.Logf"
    fi
    
    cat >> "$RESULT_FILE" << EOF

### File: \`$file:$line_num\`

**Current Code:**
\`\`\`go
$line_content
\`\`\`

**Suggested Fix:**
\`\`\`go
defer func() {
    if err := $var_name.Close(); err != nil {
        $logger("Failed to close $var_name: %v", err)
    }
}()
\`\`\`

EOF
}

# Files to analyze (excluding test files initially)
FILES=$(find . -name "*.go" -type f | grep -v "_test.go" | grep -v "vendor/" | grep -v "auto_fix")

# Analyze each file
for file in $FILES; do
    # Find lines with defer XXX.Close() that don't have error handling
    while IFS= read -r line; do
        line_num=$(echo "$line" | cut -d: -f1)
        line_content=$(echo "$line" | cut -d: -f2-)
        
        # Skip if it already has error handling
        if [[ ! "$line_content" =~ "if err :=" ]]; then
            generate_fix "$file" "$line_num" "$line_content"
        fi
    done < <(grep -n "defer.*\.Close()" "$file" 2>/dev/null)
done

# Add summary
cat >> "$RESULT_FILE" << 'EOF'

## Common Patterns

### 1. File Operations
```go
// Before:
defer file.Close()

// After:
defer func() {
    if err := file.Close(); err != nil {
        log.Printf("Failed to close file: %v", err)
    }
}()
```

### 2. HTTP Response Body
```go
// Before:
defer resp.Body.Close()

// After:
defer func() {
    if err := resp.Body.Close(); err != nil {
        log.Printf("Failed to close response body: %v", err)
    }
}()
```

### 3. Database Connections
```go
// Before:
defer db.Close()

// After:
defer func() {
    if err := db.Close(); err != nil {
        log.Printf("Failed to close database connection: %v", err)
    }
}()
```

## Automated Application

To apply these fixes semi-automatically, you can use the following approach:

1. Review each suggestion in this file
2. Use your editor's search and replace with regex
3. Or use the provided sed commands below

### Sed Commands (use with caution, review changes):

```bash
# For simple file.Close() patterns
sed -i.bak 's/defer \([a-zA-Z0-9_]*\)\.Close()/defer func() {\
    if err := \1.Close(); err != nil {\
        log.Printf("Failed to close %s: %v", "\1", err)\
    }\
}()/' filename.go

# Always review changes before committing!
```

EOF

# Count total issues
TOTAL_ISSUES=$(grep -r "defer.*\.Close()" --include="*.go" . | grep -v "_test.go" | grep -v "if err :=" | wc -l)

echo ""
echo "Analysis complete!"
echo "Total defer Close() calls needing error handling: $TOTAL_ISSUES"
echo "Detailed report saved to: $RESULT_FILE"
echo ""
echo "Next steps:"
echo "1. Review $RESULT_FILE for specific fixes"
echo "2. Apply fixes manually or using search/replace"
echo "3. Run 'make lint' to verify fixes"