package diagnostics

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCrashReporter = errors.New("invalid crashReporter options")
	ErrTracingActive        = errors.New("contentTracing is already active")
	ErrTracingInactive      = errors.New("contentTracing is not active")
)

type CrashReporterOptions struct {
	SubmitURL       string
	ProductName     string
	CompanyName     string
	UploadToServer  bool
	ExtraParameters map[string]string
}

type CrashReport struct {
	ProcessType     string
	Reason          string
	ExtraParameters map[string]string
	Stack           []string
}

type CrashReporter struct {
	options CrashReporterOptions
	keys    map[string]string
	reports []CrashReport
}

func StartCrashReporter(options CrashReporterOptions) (*CrashReporter, error) {
	normalized, err := normalizeCrashReporterOptions(options)
	if err != nil {
		return nil, err
	}
	return &CrashReporter{options: normalized, keys: make(map[string]string)}, nil
}

func (c *CrashReporter) AddExtraParameter(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, "\x00\r\n") {
		return fmt.Errorf("%w: key", ErrInvalidCrashReporter)
	}
	c.keys[key] = value
	return nil
}

func (c *CrashReporter) Capture(report CrashReport) error {
	report.ProcessType = strings.TrimSpace(report.ProcessType)
	report.Reason = strings.TrimSpace(report.Reason)
	if report.ProcessType == "" || report.Reason == "" {
		return fmt.Errorf("%w: report", ErrInvalidCrashReporter)
	}
	report.ExtraParameters = cloneMap(c.options.ExtraParameters)
	for key, value := range c.keys {
		report.ExtraParameters[key] = value
	}
	report.Stack = append([]string(nil), report.Stack...)
	c.reports = append(c.reports, report)
	return nil
}

func (c *CrashReporter) Reports() []CrashReport {
	out := make([]CrashReport, len(c.reports))
	for i, report := range c.reports {
		out[i] = report
		out[i].ExtraParameters = cloneMap(report.ExtraParameters)
		out[i].Stack = append([]string(nil), report.Stack...)
	}
	return out
}

func normalizeCrashReporterOptions(options CrashReporterOptions) (CrashReporterOptions, error) {
	options.SubmitURL = strings.TrimSpace(options.SubmitURL)
	options.ProductName = strings.TrimSpace(options.ProductName)
	options.CompanyName = strings.TrimSpace(options.CompanyName)
	if options.UploadToServer && options.SubmitURL == "" {
		return CrashReporterOptions{}, fmt.Errorf("%w: submitURL required for upload", ErrInvalidCrashReporter)
	}
	for key := range options.ExtraParameters {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\x00\r\n") {
			return CrashReporterOptions{}, fmt.Errorf("%w: extra parameter key", ErrInvalidCrashReporter)
		}
	}
	options.ExtraParameters = cloneMap(options.ExtraParameters)
	return options, nil
}

type TraceConfig struct {
	Categories    []string
	HeapProfiling bool
}

type TraceResult struct {
	Categories  []string
	HeapProfile bool
	Path        string
}

type Tracer struct {
	active  bool
	config  TraceConfig
	results []TraceResult
}

func (t *Tracer) Start(config TraceConfig) error {
	if t.active {
		return ErrTracingActive
	}
	if len(config.Categories) == 0 {
		config.Categories = []string{"*"}
	}
	config.Categories = append([]string(nil), config.Categories...)
	t.config = config
	t.active = true
	return nil
}

func (t *Tracer) Stop(path string) (TraceResult, error) {
	if !t.active {
		return TraceResult{}, ErrTracingInactive
	}
	path = strings.TrimSpace(path)
	if path == "" {
		path = "trace.json"
	}
	result := TraceResult{
		Categories:  append([]string(nil), t.config.Categories...),
		HeapProfile: t.config.HeapProfiling,
		Path:        path,
	}
	stored := result
	stored.Categories = append([]string(nil), result.Categories...)
	t.results = append(t.results, stored)
	t.active = false
	return result, nil
}

func (t *Tracer) Results() []TraceResult {
	out := make([]TraceResult, len(t.results))
	for i, result := range t.results {
		out[i] = result
		out[i].Categories = append([]string(nil), result.Categories...)
	}
	return out
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
