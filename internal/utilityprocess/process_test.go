package utilityprocess

import (
	"errors"
	"reflect"
	"testing"
)

func TestForkNormalizesOptions(t *testing.T) {
	process, err := Fork(Options{
		ModulePath:  " worker.js ",
		Args:        []string{"--flag"},
		Environment: []string{"A=B"},
		ServiceName: " service ",
		Sandbox:     true,
		DisclaimTCC: true,
	})
	if err != nil {
		t.Fatalf("Fork() error = %v", err)
	}
	options := process.Options()
	if options.ModulePath != "worker.js" || options.Stdio != StdioPipe || options.ServiceName != "service" || !options.Sandbox || !options.DisclaimTCC {
		t.Fatalf("Options() = %#v", options)
	}
	options.Args[0] = "mutated"
	if got := process.Options().Args[0]; got != "--flag" {
		t.Fatalf("Options() returned mutable args: %q", got)
	}
}

func TestForkRejectsInvalidOptions(t *testing.T) {
	cases := []Options{
		{},
		{ModulePath: "worker.js", Stdio: "socket"},
		{ModulePath: "worker.js", Environment: []string{"BAD"}},
		{ModulePath: "worker.js", Environment: []string{"A=B\nC=D"}},
	}
	for _, tc := range cases {
		if _, err := Fork(tc); !errors.Is(err, ErrInvalidOptions) {
			t.Fatalf("Fork(%#v) error = %v, want ErrInvalidOptions", tc, err)
		}
	}
}

func TestProcessLifecycleMessagesAndStdin(t *testing.T) {
	process, err := Fork(Options{ModulePath: "worker.js"})
	if err != nil {
		t.Fatalf("Fork() error = %v", err)
	}
	if err := process.PostMessage([]byte("before")); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("PostMessage(before start) error = %v, want ErrNotRunning", err)
	}
	if err := process.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := process.Start(); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("Start(again) error = %v, want ErrAlreadyRunning", err)
	}
	message := []byte("hello")
	if err := process.PostMessage(message); err != nil {
		t.Fatalf("PostMessage() error = %v", err)
	}
	message[0] = 'H'
	if err := process.WriteStdin([]byte("stdin")); err != nil {
		t.Fatalf("WriteStdin() error = %v", err)
	}
	if err := process.Kill(7); err != nil {
		t.Fatalf("Kill() error = %v", err)
	}
	wantEvents := []EventRecord{
		{Event: EventSpawn},
		{Event: EventMessage, Message: []byte("hello")},
		{Event: EventExit, ExitCode: 7},
	}
	if got := process.Events(); !reflect.DeepEqual(got, wantEvents) {
		t.Fatalf("Events() = %#v, want %#v", got, wantEvents)
	}
	writes := process.StdinWrites()
	if len(writes) != 1 || string(writes[0]) != "stdin" {
		t.Fatalf("StdinWrites() = %#v", writes)
	}
	if err := process.Kill(0); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("Kill(after exit) error = %v, want ErrNotRunning", err)
	}
}

func TestWriteStdinRequiresPipe(t *testing.T) {
	process, err := Fork(Options{ModulePath: "worker.js", Stdio: StdioIgnore})
	if err != nil {
		t.Fatalf("Fork() error = %v", err)
	}
	if err := process.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := process.WriteStdin([]byte("data")); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("WriteStdin(non-pipe) error = %v, want ErrInvalidOptions", err)
	}
}
