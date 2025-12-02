package batch

import (
	"testing"
)

func TestParser_ParseEcho(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
		wantErr bool
	}{
		{
			name:    "simple echo",
			input:   "echo hello",
			wantMsg: "hello",
		},
		{
			name:    "echo with quotes",
			input:   `echo "hello world"`,
			wantMsg: "hello world",
		},
		{
			name:    "empty echo",
			input:   "echo",
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			script, err := parser.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(script.Commands) != 1 {
					t.Errorf("expected 1 command, got %d", len(script.Commands))
					return
				}

				cmd, ok := script.Commands[0].(*EchoCommand)
				if !ok {
					t.Errorf("expected EchoCommand, got %T", script.Commands[0])
					return
				}

				if cmd.Message != tt.wantMsg {
					t.Errorf("expected message %q, got %q", tt.wantMsg, cmd.Message)
				}
			}
		})
	}
}

func TestParser_ParseExit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCode int
		wantErr  bool
	}{
		{
			name:     "exit without code",
			input:    "exit",
			wantCode: 0,
		},
		{
			name:     "exit with code",
			input:    "exit 1",
			wantCode: 1,
		},
		{
			name:    "exit with invalid code",
			input:   "exit abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			script, err := parser.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(script.Commands) != 1 {
					t.Errorf("expected 1 command, got %d", len(script.Commands))
					return
				}

				cmd, ok := script.Commands[0].(*ExitCommand)
				if !ok {
					t.Errorf("expected ExitCommand, got %T", script.Commands[0])
					return
				}

				if cmd.Code != tt.wantCode {
					t.Errorf("expected code %d, got %d", tt.wantCode, cmd.Code)
				}
			}
		})
	}
}

func TestParser_ParseCLICommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantArgs []string
	}{
		{
			name:     "command without args",
			input:    "status",
			wantName: "status",
			wantArgs: []string{},
		},
		{
			name:     "command with args",
			input:    "cache clear --type maven",
			wantName: "cache",
			wantArgs: []string{"clear", "--type", "maven"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			script, err := parser.Parse()
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			if len(script.Commands) != 1 {
				t.Errorf("expected 1 command, got %d", len(script.Commands))
				return
			}

			cmd, ok := script.Commands[0].(*CLICommand)
			if !ok {
				t.Errorf("expected CLICommand, got %T", script.Commands[0])
				return
			}

			if cmd.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, cmd.Name)
			}

			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("expected %d args, got %d", len(tt.wantArgs), len(cmd.Args))
				return
			}

			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d]: expected %q, got %q", i, tt.wantArgs[i], arg)
				}
			}
		})
	}
}

func TestParser_ParseConditional(t *testing.T) {
	input := `if last_exit == 0 then
    echo success
else
    echo failure
fi`

	parser := NewParser(input)
	script, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(script.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(script.Commands))
	}

	cmd, ok := script.Commands[0].(*ConditionalCommand)
	if !ok {
		t.Fatalf("expected ConditionalCommand, got %T", script.Commands[0])
	}

	if cmd.Condition != "last_exit == 0" {
		t.Errorf("expected condition %q, got %q", "last_exit == 0", cmd.Condition)
	}

	if len(cmd.ThenBlock) != 1 {
		t.Errorf("expected 1 command in then block, got %d", len(cmd.ThenBlock))
	}

	if len(cmd.ElseBlock) != 1 {
		t.Errorf("expected 1 command in else block, got %d", len(cmd.ElseBlock))
	}
}

func TestParser_MultipleCommands(t *testing.T) {
	input := `echo Starting
cache clear
echo Done`

	parser := NewParser(input)
	script, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(script.Commands) != 3 {
		t.Errorf("expected 3 commands, got %d", len(script.Commands))
	}
}

func TestParser_IgnoreComments(t *testing.T) {
	input := `# This is a comment
echo hello
# Another comment
echo world`

	parser := NewParser(input)
	script, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(script.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(script.Commands))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid script",
			input: `echo hello
cache clear`,
			wantErr: false,
		},
		{
			name:    "invalid exit code",
			input:   "exit abc",
			wantErr: true,
		},
		{
			name: "valid conditional",
			input: `if last_exit == 0 then
    echo success
fi`,
			wantErr: false,
		},
		{
			name: "missing fi",
			input: `if last_exit == 0 then
    echo success`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
