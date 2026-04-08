package project

import (
	"os"
	"path/filepath"
)

const (
	// PrimaryProjectInstructionFile is the preferred root project instruction file (ecosystem standard).
	PrimaryProjectInstructionFile = "AGENTS.md"
	// FallbackProjectInstructionFile is used when the primary file is absent (backward compatibility).
	FallbackProjectInstructionFile = "CLAUDE.md"
)

// InstructionFilenameInDir returns the basename of the project instruction file to use in dir:
// PrimaryProjectInstructionFile if it exists, else FallbackProjectInstructionFile if it exists,
// else "". The same logic applies at repo root and in nested directories.
func InstructionFilenameInDir(dir string) string {
	primary := filepath.Join(dir, PrimaryProjectInstructionFile)
	if _, err := os.Stat(primary); err == nil {
		return PrimaryProjectInstructionFile
	}
	fallback := filepath.Join(dir, FallbackProjectInstructionFile)
	if _, err := os.Stat(fallback); err == nil {
		return FallbackProjectInstructionFile
	}
	return ""
}

// GetProjectInstructionFile returns the preferred project instruction filename at root (AGENTS.md
// first, then CLAUDE.md). It is equivalent to InstructionFilenameInDir(root).
func GetProjectInstructionFile(root string) string {
	return InstructionFilenameInDir(root)
}

// HasProjectInstructionFile reports whether either primary or fallback instruction file exists in root.
func HasProjectInstructionFile(root string) bool {
	return InstructionFilenameInDir(root) != ""
}

// GetProjectInstructionFilePath returns the path to the instruction file that should be used:
// the existing file if either exists (preferring AGENTS.md), otherwise a path to PrimaryProjectInstructionFile
// under root (default target for init / memory creation). Root is cleaned with filepath.Clean.
func GetProjectInstructionFilePath(root string) string {
	clean := filepath.Clean(root)
	name := InstructionFilenameInDir(clean)
	if name == "" {
		name = PrimaryProjectInstructionFile
	}
	return filepath.Join(clean, name)
}

// IsProjectInstructionBasename reports whether base is a known root-level project instruction filename.
// Useful for tooling that should treat writes to these files like other project instruction edits.
func IsProjectInstructionBasename(base string) bool {
	return base == PrimaryProjectInstructionFile || base == FallbackProjectInstructionFile
}
