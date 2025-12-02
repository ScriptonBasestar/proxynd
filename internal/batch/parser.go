package batch

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Parser parses batch script files into executable BatchScript
type Parser struct {
	lines   []string
	current int
}

// NewParser creates a new parser
func NewParser(source string) *Parser {
	lines := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(source))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	return &Parser{
		lines:   lines,
		current: 0,
	}
}

// Parse parses the batch script
func (p *Parser) Parse() (*BatchScript, error) {
	commands := make([]Command, 0)

	for p.current < len(p.lines) {
		cmd, err := p.parseCommand()
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", p.current+1, err)
		}
		if cmd != nil {
			commands = append(commands, cmd)
		}
		p.current++
	}

	return &BatchScript{
		Commands: commands,
		Source:   strings.Join(p.lines, "\n"),
	}, nil
}

// parseCommand parses a single command
func (p *Parser) parseCommand() (Command, error) {
	if p.current >= len(p.lines) {
		return nil, nil
	}

	line := p.lines[p.current]
	parts := strings.Fields(line)

	if len(parts) == 0 {
		return nil, nil
	}

	keyword := parts[0]

	switch keyword {
	case "echo":
		return p.parseEcho(parts[1:])
	case "exit":
		return p.parseExit(parts[1:])
	case "if":
		return p.parseConditional()
	default:
		// Treat as CLI command
		return p.parseCLICommand(parts)
	}
}

// parseEcho parses an echo command
func (p *Parser) parseEcho(args []string) (*EchoCommand, error) {
	if len(args) == 0 {
		return &EchoCommand{Message: ""}, nil
	}

	// Join all args as message, handle quoted strings
	message := strings.Join(args, " ")
	message = strings.Trim(message, "\"")

	return &EchoCommand{Message: message}, nil
}

// parseExit parses an exit command
func (p *Parser) parseExit(args []string) (*ExitCommand, error) {
	code := 0
	if len(args) > 0 {
		var err error
		code, err = strconv.Atoi(args[0])
		if err != nil {
			return nil, fmt.Errorf("invalid exit code: %s", args[0])
		}
	}

	return &ExitCommand{Code: code}, nil
}

// parseCLICommand parses a CLI command
func (p *Parser) parseCLICommand(parts []string) (*CLICommand, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return &CLICommand{
		Name: parts[0],
		Args: parts[1:],
	}, nil
}

// parseConditional parses an if-then-else-fi block
func (p *Parser) parseConditional() (*ConditionalCommand, error) {
	line := p.lines[p.current]

	// Extract condition
	// Format: if condition then
	parts := strings.Fields(line)
	if len(parts) < 3 || parts[0] != "if" || parts[len(parts)-1] != "then" {
		return nil, fmt.Errorf("invalid if statement: %s", line)
	}

	// Condition is everything between "if" and "then"
	condition := strings.Join(parts[1:len(parts)-1], " ")

	// Parse then block
	p.current++
	thenBlock := make([]Command, 0)

	for p.current < len(p.lines) {
		line := p.lines[p.current]
		keyword := strings.Fields(line)[0]

		if keyword == "else" || keyword == "fi" {
			break
		}

		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		if cmd != nil {
			thenBlock = append(thenBlock, cmd)
		}
		p.current++
	}

	// Parse else block if present
	elseBlock := make([]Command, 0)
	if p.current < len(p.lines) && strings.Fields(p.lines[p.current])[0] == "else" {
		p.current++

		for p.current < len(p.lines) {
			line := p.lines[p.current]
			keyword := strings.Fields(line)[0]

			if keyword == "fi" {
				break
			}

			cmd, err := p.parseCommand()
			if err != nil {
				return nil, err
			}
			if cmd != nil {
				elseBlock = append(elseBlock, cmd)
			}
			p.current++
		}
	}

	// Verify we have 'fi'
	if p.current >= len(p.lines) || strings.Fields(p.lines[p.current])[0] != "fi" {
		return nil, fmt.Errorf("missing 'fi' for if statement")
	}

	return &ConditionalCommand{
		Condition: condition,
		ThenBlock: thenBlock,
		ElseBlock: elseBlock,
	}, nil
}

// Validate validates a batch script for syntax errors
func Validate(source string) error {
	parser := NewParser(source)
	_, err := parser.Parse()
	return err
}
