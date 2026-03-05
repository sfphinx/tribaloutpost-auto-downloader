// Command buildvl2 creates a VL2 (ZIP) archive from the TorqueScript source files.
// VL2 files are ZIP archives containing scripts in T2's expected directory structure.
//
// Usage: go run ./cmd/buildvl2 -root <repo-root> -output <output-path>
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// vl2Structure maps source files (relative to repo root) to their paths inside the VL2.
// T2 expects scripts under scripts/autoexec/ to be loaded automatically.
var vl2Structure = map[string]string{
	"t2script/TribalOutpostAutoDL.cs":      "scripts/autoexec/tribaloutpost/TribalOutpostAutoDL.cs",
	"t2script/gui/TribalOutpostAutoDL.gui": "scripts/autoexec/tribaloutpost/gui/TribalOutpostAutoDL.gui",
}

func main() {
	root := flag.String("root", ".", "repository root directory")
	output := flag.String("output", "TribalOutpostAutoDL.vl2", "output VL2 file path")
	flag.Parse()

	if err := buildVL2(*root, *output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Built: %s\n", *output)
}

func buildVL2(root, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for src, dst := range vl2Structure {
		srcPath := filepath.Join(root, src)
		if err := addFileToZip(w, srcPath, dst); err != nil {
			return fmt.Errorf("failed to add %s: %w", src, err)
		}
		fmt.Printf("  added: %s -> %s\n", src, dst)
	}

	return nil
}

func addFileToZip(w *zip.Writer, srcPath, archivePath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", srcPath, err)
	}

	header := &zip.FileHeader{
		Name:     archivePath,
		Method:   zip.Deflate,
		Modified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	header.SetMode(fs.FileMode(0644))

	fw, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = fw.Write(data)
	return err
}
