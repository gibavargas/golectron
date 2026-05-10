package packaging

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidManifest = errors.New("invalid packaging manifest")

type Fuse string

const (
	FuseRunAsNode              Fuse = "runAsNode"
	FuseEnableCookieEncryption Fuse = "enableCookieEncryption"
	FuseOnlyLoadAppFromAsar    Fuse = "onlyLoadAppFromAsar"
)

type PlatformPackage string

const (
	PackageDMG  PlatformPackage = "dmg"
	PackageMAS  PlatformPackage = "mas"
	PackageNSIS PlatformPackage = "nsis"
	PackageMSIX PlatformPackage = "msix"
	PackageDeb  PlatformPackage = "deb"
	PackageRPM  PlatformPackage = "rpm"
)

type Signing struct {
	Identity        string
	Entitlements    string
	HardenedRuntime bool
	Notarized       bool
}

type Manifest struct {
	AppName      string
	Executable   string
	Helpers      []string
	Resources    []string
	Fuses        map[Fuse]bool
	Signing      Signing
	Packages     []PlatformPackage
	MASCompliant bool
}

func Validate(manifest Manifest) error {
	if strings.TrimSpace(manifest.AppName) == "" {
		return fmt.Errorf("%w: app name", ErrInvalidManifest)
	}
	if strings.TrimSpace(manifest.Executable) == "" {
		return fmt.Errorf("%w: executable", ErrInvalidManifest)
	}
	if len(manifest.Resources) == 0 {
		return fmt.Errorf("%w: resources", ErrInvalidManifest)
	}
	for _, helper := range manifest.Helpers {
		if strings.TrimSpace(helper) == "" {
			return fmt.Errorf("%w: helper", ErrInvalidManifest)
		}
	}
	for fuse := range manifest.Fuses {
		if !validFuse(fuse) {
			return fmt.Errorf("%w: fuse", ErrInvalidManifest)
		}
	}
	for _, pkg := range manifest.Packages {
		if !validPackage(pkg) {
			return fmt.Errorf("%w: package", ErrInvalidManifest)
		}
		if pkg == PackageMAS && !manifest.MASCompliant {
			return fmt.Errorf("%w: MAS package requires MAS compliance", ErrInvalidManifest)
		}
		if (pkg == PackageDMG || pkg == PackageMAS || pkg == PackageMSIX) && strings.TrimSpace(manifest.Signing.Identity) == "" {
			return fmt.Errorf("%w: signed package requires identity", ErrInvalidManifest)
		}
	}
	if manifest.Signing.Notarized && (!manifest.Signing.HardenedRuntime || strings.TrimSpace(manifest.Signing.Identity) == "") {
		return fmt.Errorf("%w: notarization requires signing identity and hardened runtime", ErrInvalidManifest)
	}
	return nil
}

func Summary(manifest Manifest) (map[string]bool, error) {
	if err := Validate(manifest); err != nil {
		return nil, err
	}
	return map[string]bool{
		"hasHelpers":      len(manifest.Helpers) > 0,
		"hasResources":    len(manifest.Resources) > 0,
		"signed":          strings.TrimSpace(manifest.Signing.Identity) != "",
		"notarized":       manifest.Signing.Notarized,
		"hardenedRuntime": manifest.Signing.HardenedRuntime,
		"masCompliant":    manifest.MASCompliant,
	}, nil
}

func validFuse(fuse Fuse) bool {
	switch fuse {
	case FuseRunAsNode, FuseEnableCookieEncryption, FuseOnlyLoadAppFromAsar:
		return true
	default:
		return false
	}
}

func validPackage(pkg PlatformPackage) bool {
	switch pkg {
	case PackageDMG, PackageMAS, PackageNSIS, PackageMSIX, PackageDeb, PackageRPM:
		return true
	default:
		return false
	}
}
