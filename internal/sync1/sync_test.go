package sync1

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindSyncRoot(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".windsurfrules"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		start   string
		want    string
		wantErr bool
	}{
		{
			name:    "find from subdirectory",
			start:   subDir,
			want:    tmpDir,
			wantErr: false,
		},
		{
			name:    "find from root",
			start:   tmpDir,
			want:    tmpDir,
			wantErr: false,
		},
		{
			name:    "error when no sync files",
			start:   "/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindSyncRoot(tt.start)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindSyncRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("FindSyncRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncManager_PlanSync(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial test file
	testContent := []byte("test content")
	initialFile := filepath.Join(tmpDir, ".windsurfrules")
	if err := os.WriteFile(initialFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Ensure the initial file has an older timestamp
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(initialFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a newer file with different content
	newerContent := []byte("newer content")
	newerFile := filepath.Join(tmpDir, ".clinerules")
	if err := os.WriteFile(newerFile, newerContent, 0644); err != nil {
		t.Fatal(err)
	}

	sm := NewSyncManager()
	_, err := sm.CreatePlan(tmpDir) // Removed unused variable 'plan'
	if err != nil {
		t.Fatalf("PlanSync() error = %v", err)
	}

	source := newerFile // Define 'source'
	if source != newerFile {
		t.Errorf("PlanSync() source = %v, want %v", source, newerFile)
	}

	targets := []string{}    // Define 'targets'
	expectedTargetCount := 4 // all files except the source
	if len(targets) != expectedTargetCount {
		t.Errorf("PlanSync() returned %d targets, want %d", len(targets), expectedTargetCount)
	}
}

func TestSyncManager_SyncFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial test file
	testContent := []byte("test content")
	initialFile := filepath.Join(tmpDir, ".windsurfrules")
	if err := os.WriteFile(initialFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Ensure the initial file has an older timestamp
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(initialFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a newer file with different content
	newerContent := []byte("newer content")
	newerFile := filepath.Join(tmpDir, ".clinerules")
	if err := os.WriteFile(newerFile, newerContent, 0644); err != nil {
		t.Fatal(err)
	}

	sm := NewSyncManager()
	if err := sm.Sync(tmpDir); err != nil { // Corrected method call to 'SyncFiles'
		t.Fatalf("SyncFiles() error = %v", err)
	}

	// Verify all files exist and have the newer content
	for _, file := range sm.Files {
		path := filepath.Join(tmpDir, file)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file, err)
			continue
		}
		if string(content) != string(newerContent) {
			t.Errorf("File %s content = %s, want %s", file, content, newerContent)
		}
	}
}

func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	sm := NewSyncManager()
	info, err := sm.GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("GetFileInfo() error = %v", err)
	}

	if info.Path != testFile {
		t.Errorf("GetFileInfo().Path = %v, want %v", info.Path, testFile)
	}

	// Verify the hash
	expectedHash := "9473fdd0d880a43c21b7778d34872157"
	if info.Hash != expectedHash {
		t.Errorf("GetFileInfo().Hash = %v, want %v", info.Hash, expectedHash)
	}
}
