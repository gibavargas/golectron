package npm

import (
	"errors"
	"testing"
)

func TestPlanInstallIsManualOnly(t *testing.T) {
	plan, err := PlanInstall(InstallRequest{
		Manual: true,
		Environment: map[string]string{
			"ELECTRON_INSTALL_PLATFORM":     "linux",
			"ELECTRON_INSTALL_ARCH":         "arm64",
			"ELECTRON_SKIP_BINARY_DOWNLOAD": "1",
		},
	})
	if err != nil {
		t.Fatalf("PlanInstall() error = %v", err)
	}
	if plan.Platform != "linux" || plan.Arch != "arm64" || !plan.Download || !plan.Manual || !plan.RemovedSkipSet {
		t.Fatalf("PlanInstall() = %#v", plan)
	}
}

func TestPlanBinExecutionDownloadsLazilyWhenBinaryMissing(t *testing.T) {
	plan, err := PlanBinExecution(InstallRequest{
		BinaryExists: false,
		Environment: map[string]string{
			"ELECTRON_INSTALL_PLATFORM": "darwin",
			"ELECTRON_INSTALL_ARCH":     "x64",
		},
	})
	if err != nil {
		t.Fatalf("PlanBinExecution() error = %v", err)
	}
	if plan.Platform != "darwin" || plan.Arch != "x64" || !plan.Download || plan.Manual {
		t.Fatalf("PlanBinExecution(missing) = %#v", plan)
	}
	plan, err = PlanBinExecution(InstallRequest{BinaryExists: true})
	if err != nil {
		t.Fatalf("PlanBinExecution(existing) error = %v", err)
	}
	if plan.Download {
		t.Fatalf("PlanBinExecution(existing).Download = true, want false")
	}
}

func TestPlanInstallRejectsInvalidEnvVars(t *testing.T) {
	cases := []map[string]string{
		{"ELECTRON_INSTALL_PLATFORM": ""},
		{"ELECTRON_INSTALL_PLATFORM": "linux\nbad"},
		{"ELECTRON_INSTALL_ARCH": ""},
		{"ELECTRON_INSTALL_ARCH": "arm64\nbad"},
	}
	for _, tc := range cases {
		if _, err := PlanInstall(InstallRequest{Environment: tc}); !errors.Is(err, ErrInvalidInstall) {
			t.Fatalf("PlanInstall(%#v) error = %v, want ErrInvalidInstall", tc, err)
		}
	}
}

func TestResolveAndRequireBinary(t *testing.T) {
	path, err := ResolveBinaryPath("/pkg", "linux", "arm64")
	if err != nil {
		t.Fatalf("ResolveBinaryPath() error = %v", err)
	}
	if path != "/pkg/dist/linux-arm64/electron-go" {
		t.Fatalf("path = %q", path)
	}
	if err := RequireBinary(false); !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("RequireBinary(false) error = %v, want ErrBinaryMissing", err)
	}
	if err := RequireBinary(true); err != nil {
		t.Fatalf("RequireBinary(true) error = %v", err)
	}
}
