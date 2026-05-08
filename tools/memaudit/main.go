package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Report struct {
	Root                 string           `json:"root"`
	FilesScanned         int              `json:"files_scanned"`
	AnyUses              int              `json:"any_uses"`
	EmptyInterfaceUses   int              `json:"empty_interface_uses"`
	UnsafeImports        int              `json:"unsafe_imports"`
	SyncPoolUses         int              `json:"sync_pool_uses"`
	Hotspots             []Hotspot        `json:"hotspots,omitempty"`
	UnsafeImportDetected bool             `json:"unsafe_import_detected"`
	CEFLayout            *CEFLayoutReport `json:"cef_layout,omitempty"`
}

type Hotspot struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type CEFLayoutReport struct {
	Root    string   `json:"root"`
	Valid   bool     `json:"valid"`
	Missing []string `json:"missing,omitempty"`
	Notes   []string `json:"notes,omitempty"`
}

func main() {
	root := flag.String("root", ".", "root directory to scan")
	cefLayout := flag.String("check-cef-layout", "", "optional bin/runtime directory to validate for CEF runtime files")
	jsonOut := flag.Bool("json", false, "emit JSON instead of markdown")
	failOnUnsafe := flag.Bool("fail-on-unsafe", false, "exit nonzero when unsafe imports are found")
	flag.Parse()

	report, err := scanRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memaudit: %v\n", err)
		os.Exit(1)
	}
	if *cefLayout != "" {
		layout := checkCEFLayout(*cefLayout)
		report.CEFLayout = &layout
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "memaudit: encode: %v\n", err)
			os.Exit(1)
		}
	} else {
		printMarkdown(report)
	}

	if *failOnUnsafe && report.UnsafeImportDetected {
		os.Exit(3)
	}
	if report.CEFLayout != nil && !report.CEFLayout.Valid {
		os.Exit(4)
	}
}

func scanRoot(root string) (Report, error) {
	report := Report{Root: root}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		fileReport, err := scanFile(path)
		if err != nil {
			return err
		}
		report.FilesScanned++
		report.AnyUses += fileReport.AnyUses
		report.EmptyInterfaceUses += fileReport.EmptyInterfaceUses
		report.UnsafeImports += fileReport.UnsafeImports
		report.SyncPoolUses += fileReport.SyncPoolUses
		report.Hotspots = append(report.Hotspots, fileReport.Hotspots...)
		return nil
	})
	report.UnsafeImportDetected = report.UnsafeImports > 0
	return report, err
}

type fileReport struct {
	AnyUses            int
	EmptyInterfaceUses int
	UnsafeImports      int
	SyncPoolUses       int
	Hotspots           []Hotspot
}

func scanFile(path string) (fileReport, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return fileReport{}, fmt.Errorf("parse %s: %w", path, err)
	}

	var report fileReport
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "unsafe" {
			report.UnsafeImports++
			report.Hotspots = append(report.Hotspots, hotspot(fset, imp.Pos(), path, "unsafe-import", "import unsafe"))
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Ident:
			if n.Name == "any" {
				report.AnyUses++
				report.Hotspots = append(report.Hotspots, hotspot(fset, n.Pos(), path, "any", "any"))
			}
		case *ast.InterfaceType:
			if n.Methods == nil || len(n.Methods.List) == 0 {
				report.EmptyInterfaceUses++
				report.Hotspots = append(report.Hotspots, hotspot(fset, n.Pos(), path, "empty-interface", "interface{}"))
			}
		case *ast.SelectorExpr:
			if ident, ok := n.X.(*ast.Ident); ok && ident.Name == "sync" && n.Sel.Name == "Pool" {
				report.SyncPoolUses++
				report.Hotspots = append(report.Hotspots, hotspot(fset, n.Pos(), path, "sync-pool", "sync.Pool"))
			}
		}
		return true
	})

	return report, nil
}

func hotspot(fset *token.FileSet, pos token.Pos, file, kind, text string) Hotspot {
	position := fset.Position(pos)
	return Hotspot{File: file, Line: position.Line, Kind: kind, Text: text}
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "bin", "dist", "build", "node_modules", ".electron-go":
		return true
	default:
		return false
	}
}

func checkCEFLayout(root string) CEFLayoutReport {
	report := CEFLayoutReport{Root: root}
	requiredFiles := []string{
		"libcef.so",
		"icudtl.dat",
		"chrome_100_percent.pak",
		"resources.pak",
	}
	for _, name := range requiredFiles {
		if !regularFile(filepath.Join(root, name)) {
			report.Missing = append(report.Missing, name)
		}
	}
	if !regularFile(filepath.Join(root, "v8_context_snapshot.bin")) &&
		!regularFile(filepath.Join(root, "v8_context_snapshot_blob.bin")) &&
		!regularFile(filepath.Join(root, "snapshot_blob.bin")) {
		report.Missing = append(report.Missing, "v8_context_snapshot.bin or snapshot_blob.bin")
	}
	locales := filepath.Join(root, "locales")
	if !directory(locales) {
		report.Missing = append(report.Missing, "locales/")
	} else if !hasPAKFile(locales) {
		report.Missing = append(report.Missing, "locales/*.pak")
	}
	report.Valid = len(report.Missing) == 0
	if report.Valid {
		report.Notes = append(report.Notes, "CEF runtime layout is present")
	}
	return report
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func directory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasPAKFile(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pak") {
			return true
		}
	}
	return false
}

func printMarkdown(report Report) {
	fmt.Printf("# Electron-Go Memory Audit\n\n")
	fmt.Printf("- Root: `%s`\n", report.Root)
	fmt.Printf("- Go files scanned: `%d`\n", report.FilesScanned)
	fmt.Printf("- `any` uses: `%d`\n", report.AnyUses)
	fmt.Printf("- `interface{}` uses: `%d`\n", report.EmptyInterfaceUses)
	fmt.Printf("- `unsafe` imports: `%d`\n", report.UnsafeImports)
	fmt.Printf("- `sync.Pool` uses: `%d`\n\n", report.SyncPoolUses)
	if report.CEFLayout != nil {
		fmt.Printf("## CEF Layout\n\n")
		fmt.Printf("- Root: `%s`\n", report.CEFLayout.Root)
		fmt.Printf("- Valid: `%t`\n", report.CEFLayout.Valid)
		for _, missing := range report.CEFLayout.Missing {
			fmt.Printf("- Missing: `%s`\n", missing)
		}
		fmt.Println()
	}

	if len(report.Hotspots) == 0 {
		fmt.Println("No hotspots found.")
		return
	}
	fmt.Println("## Hotspots")
	for _, item := range report.Hotspots {
		fmt.Printf("- `%s:%d` `%s` %s\n", item.File, item.Line, item.Kind, item.Text)
	}
}
