//go:build linux

package native

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type procInfo struct {
	pid  int
	ppid int
	args []string
}

type processModelSampler struct {
	rootPID  int
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	seen     map[int]ObservedProcess
}

func currentOSThreadID() int {
	return syscall.Gettid()
}

func startProcessModelSampler(rootPID int, interval time.Duration) *processModelSampler {
	if interval <= 0 {
		interval = 5 * time.Millisecond
	}
	sampler := &processModelSampler{
		rootPID:  rootPID,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		seen:     make(map[int]ObservedProcess),
	}
	go sampler.run()
	return sampler
}

func (s *processModelSampler) stopAndReport() []ObservedProcess {
	close(s.stop)
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return sortedObservedProcesses(s.seen)
}

func (s *processModelSampler) run() {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.sample()
	for {
		select {
		case <-ticker.C:
			s.sample()
		case <-s.stop:
			s.sample()
			return
		}
	}
}

func (s *processModelSampler) sample() {
	processes, err := listDescendantProcesses(s.rootPID)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, process := range processes {
		s.seen[process.PID] = process
	}
}

func listDescendantProcesses(rootPID int) ([]ObservedProcess, error) {
	procs, err := readProcTable()
	if err != nil {
		return nil, err
	}
	children := make(map[int][]procInfo)
	for _, proc := range procs {
		children[proc.ppid] = append(children[proc.ppid], proc)
	}
	seen := make(map[int]ObservedProcess)
	var walk func(int)
	walk = func(parent int) {
		for _, child := range children[parent] {
			if _, ok := seen[child.pid]; ok {
				continue
			}
			processType := CEFProcessTypeFromArgs(child.args)
			if processType == "" && len(child.args) == 0 {
				processType = "cef-child"
			}
			seen[child.pid] = ObservedProcess{
				PID:         child.pid,
				PPID:        child.ppid,
				Type:        processType,
				CommandLine: child.args,
			}
			walk(child.pid)
		}
	}
	walk(rootPID)
	return sortedObservedProcesses(seen), nil
}

func sortedObservedProcesses(processes map[int]ObservedProcess) []ObservedProcess {
	out := make([]ObservedProcess, 0, len(processes))
	for _, process := range processes {
		out = append(out, process)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PID < out[j].PID
	})
	return out
}

func readProcTable() ([]procInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	procs := make([]procInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		proc, err := readProcInfo(pid)
		if err != nil {
			continue
		}
		procs = append(procs, proc)
	}
	return procs, nil
}

func readProcInfo(pid int) (procInfo, error) {
	ppid, err := readProcPPID(pid)
	if err != nil {
		return procInfo{}, err
	}
	args, err := readProcCmdline(pid)
	if err != nil {
		return procInfo{}, err
	}
	return procInfo{pid: pid, ppid: ppid, args: args}, nil
}

func readProcPPID(pid int) (int, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, err
	}
	text := string(data)
	endName := strings.LastIndex(text, ")")
	if endName < 0 || endName+2 >= len(text) {
		return 0, os.ErrInvalid
	}
	fields := strings.Fields(text[endName+2:])
	if len(fields) < 2 {
		return 0, os.ErrInvalid
	}
	return strconv.Atoi(fields[1])
}

func readProcCmdline(pid int) ([]string, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return nil, err
	}
	data = bytes.TrimRight(data, "\x00")
	if len(data) == 0 {
		return nil, nil
	}
	parts := bytes.Split(data, []byte{0})
	args := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		args = append(args, string(part))
	}
	return args, nil
}

func reapExitedChildren(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if pid > 0 {
			continue
		}
		if err == syscall.ECHILD {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
