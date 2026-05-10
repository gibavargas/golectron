package packaging

import (
	"errors"
	"testing"
)

func TestValidateManifestAndSummary(t *testing.T) {
	manifest := Manifest{
		AppName:      "Electron Go",
		Executable:   "Electron Go.app/Contents/MacOS/Electron Go",
		Helpers:      []string{"Electron Go Helper.app"},
		Resources:    []string{"app.asar"},
		Fuses:        map[Fuse]bool{FuseRunAsNode: false, FuseOnlyLoadAppFromAsar: true},
		Signing:      Signing{Identity: "Developer ID", Entitlements: "entitlements.plist", HardenedRuntime: true, Notarized: true},
		Packages:     []PlatformPackage{PackageDMG},
		MASCompliant: false,
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	summary, err := Summary(manifest)
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if !summary["hasHelpers"] || !summary["hasResources"] || !summary["signed"] || !summary["notarized"] || !summary["hardenedRuntime"] {
		t.Fatalf("Summary() = %#v", summary)
	}
}

func TestValidateRejectsInvalidManifest(t *testing.T) {
	valid := Manifest{
		AppName:    "App",
		Executable: "App",
		Resources:  []string{"app.asar"},
		Packages:   []PlatformPackage{PackageDeb},
	}
	cases := []Manifest{
		{},
		with(valid, func(m *Manifest) { m.Executable = "" }),
		with(valid, func(m *Manifest) { m.Resources = nil }),
		with(valid, func(m *Manifest) { m.Helpers = []string{""} }),
		with(valid, func(m *Manifest) { m.Fuses = map[Fuse]bool{"unknown": true} }),
		with(valid, func(m *Manifest) { m.Packages = []PlatformPackage{"pkg"} }),
		with(valid, func(m *Manifest) {
			m.Packages = []PlatformPackage{PackageMAS}
			m.MASCompliant = false
			m.Signing.Identity = "identity"
		}),
		with(valid, func(m *Manifest) { m.Packages = []PlatformPackage{PackageDMG}; m.Signing.Identity = "" }),
		with(valid, func(m *Manifest) {
			m.Signing.Notarized = true
			m.Signing.HardenedRuntime = false
			m.Signing.Identity = "identity"
		}),
	}
	for _, tc := range cases {
		if err := Validate(tc); !errors.Is(err, ErrInvalidManifest) {
			t.Fatalf("Validate(%#v) error = %v, want ErrInvalidManifest", tc, err)
		}
	}
}

func TestValidateMASPackage(t *testing.T) {
	manifest := Manifest{
		AppName:      "App",
		Executable:   "App",
		Resources:    []string{"app.asar"},
		Signing:      Signing{Identity: "Apple Distribution"},
		Packages:     []PlatformPackage{PackageMAS},
		MASCompliant: true,
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate(MAS) error = %v", err)
	}
}

func with(manifest Manifest, edit func(*Manifest)) Manifest {
	edit(&manifest)
	return manifest
}
