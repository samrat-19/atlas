package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atlas/internal/collector"
)

func writeReports(result collector.Result, report string) {
	const outputDirectory = "output"
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return
	}

	name := reportName(result.Root)
	timestamp := time.Now().Format("20060102T150405")
	textPath := filepath.Join(outputDirectory, fmt.Sprintf("%s-%s.txt", name, timestamp))
	_ = os.WriteFile(textPath, []byte(report), 0644)

	jsonReport, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}
	jsonPath := filepath.Join(outputDirectory, fmt.Sprintf("%s-%s.json", name, timestamp))
	_ = os.WriteFile(jsonPath, jsonReport, 0644)
}

func reportName(root string) string {
	name := strings.ReplaceAll(root, string(os.PathSeparator), "_")
	return strings.ReplaceAll(name, ":", "")
}
