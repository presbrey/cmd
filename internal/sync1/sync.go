package sync1

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NewSyncManager creates a new SyncManager with the specified files to sync
func NewSyncManager() *SyncManager {
	return &SyncManager{
		Files: []string{
			"CONVENTIONS.md",
			".clinerules",
			".cursorrules",
			".github/copilot-instructions.md",
			".windsurfrules",
		},
	}
}

// SyncManager handles file synchronization operations
type SyncManager struct {
	Files []string
}

// Plan represents a synchronization plan
type Plan struct {
	SourcePath  string
	TargetPaths []string
}

// FindSyncRoot locates the root directory by searching for any of the sync files
func FindSyncRoot(startPath string) (string, error) {
	if startPath == "" {
		var err error
		startPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	sm := NewSyncManager()
	current := startPath
	for {
		// Check if any of the sync files exist in the current directory
		for _, file := range sm.Files {
			if _, err := os.Stat(filepath.Join(current, file)); err == nil {
				return current, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("no sync files found in path hierarchy")
		}
		current = parent
	}
}

// calculateMD5 computes the MD5 hash of a file
func calculateMD5(path string) (string, error) {
	var by []byte
	by, _ = os.ReadFile(path)

	hash := md5.New()
	if _, err := hash.Write(by); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// FileInfo stores information about a synchronized file
type FileInfo struct {
	Name    string
	Mode    os.FileMode
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
}

// GetFileInfo retrieves modification time and MD5 hash for a file
func (sm *SyncManager) GetFileInfo(path string) (*FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	hash, err := calculateMD5(path)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:    path,
		ModTime: stat.ModTime(),
		Hash:    hash,
		Size:    stat.Size(),
		Name:    stat.Name(),
		Mode:    stat.Mode(),
	}, nil
}

// CreatePlan returns a Plan for synchronization
func (sm *SyncManager) CreatePlan(rootPath string) (*Plan, error) {
	var latest *FileInfo
	var latestPath string

	stats := make(map[string]*FileInfo)

	// Find the most recently modified file
	for _, file := range sm.Files {
		fullPath := filepath.Join(rootPath, file)
		info, err := sm.GetFileInfo(fullPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error checking file %s: %w", file, err)
			}
			continue
		}

		stats[fullPath] = info

		if latest == nil || info.ModTime.After(latest.ModTime) {
			latest = info
			latestPath = fullPath
		}
	}

	if latest == nil {
		return nil, errors.New("no valid files found to sync")
	}

	// Collect target files that do not have the same hash as the latest file
	targets := make([]string, 0, len(sm.Files)-1)
	for _, file := range sm.Files {
		fullPath := filepath.Join(rootPath, file)
		if fullPath == latestPath {
			continue
		}
		_, exists := stats[fullPath]

		// Skip symlinks
		if exists && stats[fullPath].Mode&os.ModeSymlink == os.ModeSymlink {
			continue
		}

		// Calculate hash for target file
		targetHash, err := calculateMD5(fullPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to calculate hash for target file %s: %w", fullPath, err)
			}
			continue
		}

		// Add target file to plan if hashes do not match
		if targetHash != latest.Hash {
			targets = append(targets, fullPath)
		}
	}

	return &Plan{
		SourcePath:  latestPath,
		TargetPaths: targets,
	}, nil
}

// Sync synchronizes all target files based on the source file in the plan
func (p *Plan) Sync() error {
	// If there are no target files to update, do nothing
	if len(p.TargetPaths) == 0 {
		return nil
	}

	// Read the content of the source file
	content, err := os.ReadFile(p.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Update all target files
	for _, targetPath := range p.TargetPaths {
		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
	}

	return nil
}

// Sync creates a plan and synchronizes files based on the plan
func (sm *SyncManager) Sync(rootPath string) error {
	plan, err := sm.CreatePlan(rootPath)
	if err != nil {
		return err
	}
	return plan.Sync()
}
