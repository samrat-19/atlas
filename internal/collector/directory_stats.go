package collector

import (
	"path/filepath"
	"strings"

	"atlas/internal/model"
)

type dirStat struct {
	FileCount     int
	EvidenceCount int

	// EvidenceConfidenceSum is the running total of every matched evidence
	// item's Confidence (see model.EvidenceItem). Dividing by
	// EvidenceCount gives the directory's average evidence confidence —
	// the basis for ModuleCandidate.EvidenceStrength and, in turn,
	// NoiseProbability and BoundaryConfidence (see modules.go and
	// module_scoring.go).
	EvidenceConfidenceSum float64

	Extensions         map[string]int
	EvidenceByCategory map[string]int
	EvidenceByFilename map[string]int
}

func newDirStat() *dirStat {
	return &dirStat{
		Extensions:         make(map[string]int),
		EvidenceByCategory: make(map[string]int),
		EvidenceByFilename: make(map[string]int),
	}
}

func directoryStatsFor(stats map[string]*dirStat, relPath string) *dirStat {
	dir := filepath.Dir(relPath)
	if dir == "." {
		dir = "_root"
	}

	entry, ok := stats[dir]
	if !ok {
		entry = newDirStat()
		stats[dir] = entry
	}
	return entry
}

func (s *dirStat) updateWithFile(extension string) {
	s.FileCount++
	s.Extensions[extension]++
}

func (s *dirStat) updateWithEvidence(item model.EvidenceItem) {
	s.EvidenceCount++
	s.EvidenceConfidenceSum += item.Confidence
	s.EvidenceByCategory[item.Category]++
	s.EvidenceByFilename[item.Filename]++
}

func topLevelDirectoryForRelativePath(relPath string) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) == 0 || parts[0] == "" || len(parts) == 1 {
		return "_root"
	}
	return parts[0]
}
