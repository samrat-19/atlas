package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atlas/internal/model"
)

// writeReports writes a timestamped text report and a JSON snapshot to the
// output directory. It returns the first error encountered so the caller can
// decide whether to surface it and with what exit code.
//
// The output directory is created if it does not exist. Partial files may be
// left on disk if the JSON write succeeds but the text write fails or vice
// versa — this is acceptable because these are diagnostic artifacts, not
// transactional data.
func writeReports(result model.Result, report string) error {
	const outputDirectory = "output"
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	name := reportName(result.Root)
	timestamp := time.Now().Format("20060102T150405")

	textPath := filepath.Join(outputDirectory, fmt.Sprintf("%s-%s.txt", name, timestamp))
	if err := os.WriteFile(textPath, []byte(report), 0644); err != nil {
		return fmt.Errorf("write text report %s: %w", textPath, err)
	}

	jsonReport, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON report: %w", err)
	}
	jsonPath := filepath.Join(outputDirectory, fmt.Sprintf("%s-%s.json", name, timestamp))
	if err := os.WriteFile(jsonPath, jsonReport, 0644); err != nil {
		return fmt.Errorf("write JSON report %s: %w", jsonPath, err)
	}

	return nil
}

func reportName(root string) string {
	name := strings.ReplaceAll(root, string(os.PathSeparator), "_")
	return strings.ReplaceAll(name, ":", "")
}
