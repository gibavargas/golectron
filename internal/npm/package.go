package npm

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var (
	ErrInvalidInstall = errors.New("invalid npm install request")
	ErrBinaryMissing  = errors.New("electron-go binary is not installed")
)

type InstallRequest struct {
	Environment  map[string]string
	BinaryExists bool
	Manual       bool
}

type InstallPlan struct {
	Platform       string
	Arch           string
	Download       bool
	Manual         bool
	RemovedSkipSet bool
}

func PlanInstall(req InstallRequest) (InstallPlan, error) {
	platform := envOrDefault(req.Environment, "ELECTRON_INSTALL_PLATFORM", runtime.GOOS)
	arch := envOrDefault(req.Environment, "ELECTRON_INSTALL_ARCH", runtime.GOARCH)
	if strings.TrimSpace(platform) == "" || strings.ContainsAny(platform, "\x00\r\n") {
		return InstallPlan{}, fmt.Errorf("%w: platform", ErrInvalidInstall)
	}
	if strings.TrimSpace(arch) == "" || strings.ContainsAny(arch, "\x00\r\n") {
		return InstallPlan{}, fmt.Errorf("%w: arch", ErrInvalidInstall)
	}
	_, removedSkipSet := req.Environment["ELECTRON_SKIP_BINARY_DOWNLOAD"]
	return InstallPlan{
		Platform:       strings.TrimSpace(platform),
		Arch:           strings.TrimSpace(arch),
		Download:       req.Manual,
		Manual:         req.Manual,
		RemovedSkipSet: removedSkipSet,
	}, nil
}

func PlanBinExecution(req InstallRequest) (InstallPlan, error) {
	plan, err := PlanInstall(req)
	if err != nil {
		return InstallPlan{}, err
	}
	plan.Download = !req.BinaryExists
	plan.Manual = false
	return plan, nil
}

func ResolveBinaryPath(root, platform, arch string) (string, error) {
	root = strings.TrimSpace(root)
	platform = strings.TrimSpace(platform)
	arch = strings.TrimSpace(arch)
	if root == "" || platform == "" || arch == "" {
		return "", fmt.Errorf("%w: binary path", ErrInvalidInstall)
	}
	return root + "/dist/" + platform + "-" + arch + "/electron-go", nil
}

func RequireBinary(binaryExists bool) error {
	if !binaryExists {
		return ErrBinaryMissing
	}
	return nil
}

func envOrDefault(env map[string]string, key, fallback string) string {
	if value, ok := env[key]; ok {
		return value
	}
	return fallback
}
