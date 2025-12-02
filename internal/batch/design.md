# Batch Job System Design

## Overview

The Batch Job System allows automation of complex management tasks through script execution with support for conditional logic, error handling, and progress tracking.

## Architecture

```
┌─────────────────────────────────────────────────┐
│           CLI Commands (proxyndctl)             │
│  batch run | validate | history | cancel        │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│              Batch Executor                      │
│  - Job orchestration                            │
│  - Context management                           │
│  - Progress tracking                            │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│              Script Parser                       │
│  - Tokenization                                 │
│  - AST generation                               │
│  - Validation                                   │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│          Command Executor                        │
│  - Execute CLI commands                         │
│  - Capture output                               │
│  - Handle errors                                │
└─────────────────────────────────────────────────┘
```

## Core Components

### 1. Script Parser
**Location**: `internal/batch/parser/`

Parses `.batch` files into executable AST.

**Supported Syntax**:
```bash
# Comments
echo "message"                    # Echo commands
command arg1 arg2                 # CLI commands
if condition then ... else ... fi # Conditionals
exit N                            # Exit with code
```

### 2. Job Executor
**Location**: `internal/batch/executor/`

Executes parsed batch jobs with:
- Context cancellation support
- Progress tracking
- Error handling and rollback
- Variable interpolation

### 3. History Manager
**Location**: `internal/batch/history/`

Tracks job execution:
```go
type JobHistory struct {
    ID        string
    Script    string
    StartTime time.Time
    EndTime   time.Time
    Status    JobStatus
    Output    []string
    ExitCode  int
}
```

### 4. CLI Commands
**Location**: `cmd/proxyndctl/commands/batch/`

Commands:
- `batch run <file>` - Execute batch script
- `batch validate <file>` - Validate syntax
- `batch history` - Show execution history
- `batch cancel <job-id>` - Cancel running job

## Implementation Plan

### Phase 1: Core Parser (Day 1)
- [ ] Lexer for tokenization
- [ ] Parser for AST generation
- [ ] Basic command support
- [ ] Validation logic

### Phase 2: Executor (Day 2)
- [ ] Job executor with context
- [ ] Command execution
- [ ] Progress tracking
- [ ] Error handling

### Phase 3: CLI Integration (Day 3)
- [ ] CLI commands implementation
- [ ] History management
- [ ] Job cancellation

### Phase 4: Advanced Features (Day 4-5)
- [ ] Conditional execution
- [ ] Variable support
- [ ] Comprehensive testing
- [ ] Documentation

## Data Structures

### BatchScript
```go
type BatchScript struct {
    Commands []Command
}

type Command interface {
    Execute(ctx context.Context, executor *Executor) error
}

type EchoCommand struct {
    Message string
}

type CLICommand struct {
    Name string
    Args []string
}

type ConditionalCommand struct {
    Condition string
    ThenBlock []Command
    ElseBlock []Command
}
```

### JobExecution
```go
type JobExecution struct {
    ID          string
    Script      *BatchScript
    Context     context.Context
    Cancel      context.CancelFunc
    StartTime   time.Time
    Status      JobStatus
    CurrentStep int
    Output      *bytes.Buffer
}

type JobStatus int

const (
    JobPending JobStatus = iota
    JobRunning
    JobSuccess
    JobFailed
    JobCancelled
)
```

## File Structure

```
internal/batch/
├── parser/
│   ├── lexer.go         # Tokenization
│   ├── parser.go        # AST generation
│   ├── validator.go     # Syntax validation
│   └── ast.go           # AST node definitions
├── executor/
│   ├── executor.go      # Job execution
│   ├── command.go       # Command implementations
│   └── context.go       # Context management
├── history/
│   ├── manager.go       # History tracking
│   └── storage.go       # Persistent storage
└── types.go             # Shared types

cmd/proxyndctl/commands/batch/
├── run.go               # batch run command
├── validate.go          # batch validate command
├── history.go           # batch history command
└── cancel.go            # batch cancel command

tests/unit/batch/
├── parser_test.go
├── executor_test.go
└── integration_test.go

examples/batch/
├── maintenance.batch    # Example maintenance script
├── cache-cleanup.batch  # Cache cleanup example
└── backup.batch         # Backup example
```

## Example Batch Script

```bash
# maintenance.batch
echo "Starting maintenance..."

# Cache cleanup
cache clear --older-than 30d --force

# Check result
if last_exit == 0 then
    echo "Cache cleared successfully"
else
    echo "Cache cleanup failed"
    exit 1
fi

# Maven backup
maven-backup create --target /backup/maven

echo "Maintenance completed"
```

## Security Considerations

1. **Command Validation**: Whitelist allowed commands
2. **Path Sanitization**: Prevent directory traversal
3. **Resource Limits**: Set execution timeouts
4. **Privilege Separation**: Run with minimal permissions

## Testing Strategy

1. **Unit Tests**: Parser, executor components
2. **Integration Tests**: Full script execution
3. **Error Tests**: Failure scenarios
4. **Performance Tests**: Large scripts, concurrent jobs

## Future Enhancements

- Loop support (for/while)
- Functions/subroutines
- Remote execution
- Parallel command execution
- Script templates
