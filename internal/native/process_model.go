package native

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type ObservedProcess struct {
	PID         int      `json:"pid"`
	PPID        int      `json:"ppid"`
	Type        string   `json:"type,omitempty"`
	CommandLine []string `json:"command_line,omitempty"`
}

type RuntimeProcessModelReport struct {
	PID                  int               `json:"pid"`
	MainThreadIDBefore   int               `json:"main_thread_id_before,omitempty"`
	MainThreadIDAfter    int               `json:"main_thread_id_after,omitempty"`
	MainThreadStable     bool              `json:"main_thread_stable"`
	ContextInitialized   bool              `json:"context_initialized"`
	BrowserID            int64             `json:"browser_id"`
	MainFrameLoaded      bool              `json:"main_frame_loaded"`
	WindowClosed         bool              `json:"window_closed"`
	NoSandbox            bool              `json:"no_sandbox"`
	ObservedDescendants  []ObservedProcess `json:"observed_descendants,omitempty"`
	ObservedProcessTypes []string          `json:"observed_process_types,omitempty"`
	NoZombieDescendants  bool              `json:"no_zombie_descendants"`
	ZombieDescendants    []ObservedProcess `json:"zombie_descendants,omitempty"`
	RemainingKnownGaps   []string          `json:"remaining_known_gaps,omitempty"`
}

func CEFProcessTypeFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--type" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
		if value, ok := strings.CutPrefix(arg, "--type="); ok {
			return strings.TrimSpace(value)
		}
	}
	if len(args) > 0 {
		name := filepath.Base(args[0])
		if strings.Contains(name, "crashpad") {
			return "crashpad"
		}
	}
	return ""
}

func ProcessTypesFromObservations(processes []ObservedProcess) []string {
	seen := make(map[string]struct{})
	for _, process := range processes {
		if process.Type == "" {
			continue
		}
		seen[process.Type] = struct{}{}
	}
	types := make([]string, 0, len(seen))
	for processType := range seen {
		types = append(types, processType)
	}
	sort.Strings(types)
	return types
}

func ValidateRuntimeProcessModelReport(report RuntimeProcessModelReport) error {
	if report.PID <= 0 {
		return fmt.Errorf("process model PID is required")
	}
	if !report.MainThreadStable {
		return fmt.Errorf("CEF browser process did not stay on one locked OS thread")
	}
	if !report.ContextInitialized {
		return fmt.Errorf("CEF browser-process context was not initialized")
	}
	if report.BrowserID <= 0 {
		return fmt.Errorf("CEF BrowserWindow was not created")
	}
	if !report.MainFrameLoaded {
		return fmt.Errorf("CEF BrowserWindow main frame did not load")
	}
	if !report.WindowClosed {
		return fmt.Errorf("CEF BrowserWindow close was not observed")
	}
	if len(report.ObservedProcessTypes) == 0 {
		return fmt.Errorf("no CEF child process types were observed")
	}
	if !report.NoZombieDescendants {
		return fmt.Errorf("CEF subprocess descendants remained after shutdown")
	}
	return nil
}
