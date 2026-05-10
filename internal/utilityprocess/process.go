package utilityprocess

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidOptions = errors.New("invalid utilityProcess options")
	ErrNotRunning     = errors.New("utilityProcess is not running")
	ErrAlreadyRunning = errors.New("utilityProcess is already running")
)

type Stdio string

const (
	StdioPipe    Stdio = "pipe"
	StdioIgnore  Stdio = "ignore"
	StdioInherit Stdio = "inherit"
)

type Options struct {
	ModulePath  string
	Args        []string
	Environment []string
	Stdio       Stdio
	ServiceName string
	Sandbox     bool
	DisclaimTCC bool
}

type Event string

const (
	EventSpawn   Event = "spawn"
	EventMessage Event = "message"
	EventExit    Event = "exit"
)

type EventRecord struct {
	Event    Event
	Message  []byte
	ExitCode int
}

type Process struct {
	options Options
	running bool
	events  []EventRecord
	stdin   [][]byte
}

func Fork(options Options) (*Process, error) {
	normalized, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}
	return &Process{options: normalized}, nil
}

func (p *Process) Options() Options {
	options := p.options
	options.Args = append([]string(nil), options.Args...)
	options.Environment = append([]string(nil), options.Environment...)
	return options
}

func (p *Process) Start() error {
	if p.running {
		return ErrAlreadyRunning
	}
	p.running = true
	p.events = append(p.events, EventRecord{Event: EventSpawn})
	return nil
}

func (p *Process) PostMessage(message []byte) error {
	if !p.running {
		return ErrNotRunning
	}
	p.events = append(p.events, EventRecord{Event: EventMessage, Message: append([]byte(nil), message...)})
	return nil
}

func (p *Process) WriteStdin(data []byte) error {
	if !p.running {
		return ErrNotRunning
	}
	if p.options.Stdio != StdioPipe {
		return fmt.Errorf("%w: stdin is not piped", ErrInvalidOptions)
	}
	p.stdin = append(p.stdin, append([]byte(nil), data...))
	return nil
}

func (p *Process) Kill(exitCode int) error {
	if !p.running {
		return ErrNotRunning
	}
	p.running = false
	p.events = append(p.events, EventRecord{Event: EventExit, ExitCode: exitCode})
	return nil
}

func (p *Process) Events() []EventRecord {
	out := make([]EventRecord, len(p.events))
	for i, event := range p.events {
		out[i] = event
		out[i].Message = append([]byte(nil), event.Message...)
	}
	return out
}

func (p *Process) StdinWrites() [][]byte {
	out := make([][]byte, len(p.stdin))
	for i, write := range p.stdin {
		out[i] = append([]byte(nil), write...)
	}
	return out
}

func normalizeOptions(options Options) (Options, error) {
	options.ModulePath = strings.TrimSpace(options.ModulePath)
	options.ServiceName = strings.TrimSpace(options.ServiceName)
	if options.ModulePath == "" {
		return Options{}, fmt.Errorf("%w: module path is required", ErrInvalidOptions)
	}
	if options.Stdio == "" {
		options.Stdio = StdioPipe
	}
	if options.Stdio != StdioPipe && options.Stdio != StdioIgnore && options.Stdio != StdioInherit {
		return Options{}, fmt.Errorf("%w: stdio", ErrInvalidOptions)
	}
	options.Args = append([]string(nil), options.Args...)
	options.Environment = append([]string(nil), options.Environment...)
	for _, env := range options.Environment {
		if !strings.Contains(env, "=") || strings.ContainsAny(env, "\x00\r\n") {
			return Options{}, fmt.Errorf("%w: environment", ErrInvalidOptions)
		}
	}
	return options, nil
}
