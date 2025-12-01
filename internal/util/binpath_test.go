package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsExecutable(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		setupFile  func(string) (string, error)
		wantResult bool
	}{
		{
			name: "executable file on unix",
			setupFile: func(dir string) (string, error) {
				path := filepath.Join(dir, "test_executable")
				f, err := os.Create(path)
				if err != nil {
					return "", err
				}
				f.Close()
				// Set executable permission
				if runtime.GOOS != "windows" {
					err = os.Chmod(path, 0755)
				}
				return path, err
			},
			wantResult: true,
		},
		{
			name: "non-executable file on unix",
			setupFile: func(dir string) (string, error) {
				path := filepath.Join(dir, "test_not_executable")
				f, err := os.Create(path)
				if err != nil {
					return "", err
				}
				f.Close()
				// Set non-executable permission
				if runtime.GOOS != "windows" {
					err = os.Chmod(path, 0644)
				}
				return path, err
			},
			wantResult: runtime.GOOS == "windows", // Windows doesn't check permissions
		},
		{
			name: "non-existent file",
			setupFile: func(dir string) (string, error) {
				return filepath.Join(dir, "does_not_exist"), nil
			},
			wantResult: false,
		},
		{
			name: "directory instead of file",
			setupFile: func(dir string) (string, error) {
				path := filepath.Join(dir, "test_directory")
				err := os.Mkdir(path, 0755)
				return path, err
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.setupFile(tmpDir)
			if err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			got := isExecutable(path)
			if got != tt.wantResult {
				t.Errorf("isExecutable(%q) = %v, want %v", path, got, tt.wantResult)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		setupFile  func(string) (string, error)
		wantResult bool
	}{
		{
			name: "existing regular file",
			setupFile: func(dir string) (string, error) {
				path := filepath.Join(dir, "test_file")
				f, err := os.Create(path)
				if err != nil {
					return "", err
				}
				f.Close()
				return path, nil
			},
			wantResult: true,
		},
		{
			name: "non-existent file",
			setupFile: func(dir string) (string, error) {
				return filepath.Join(dir, "does_not_exist"), nil
			},
			wantResult: false,
		},
		{
			name: "directory",
			setupFile: func(dir string) (string, error) {
				path := filepath.Join(dir, "test_dir")
				err := os.Mkdir(path, 0755)
				return path, err
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.setupFile(tmpDir)
			if err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			got := fileExists(path)
			if got != tt.wantResult {
				t.Errorf("fileExists(%q) = %v, want %v", path, got, tt.wantResult)
			}
		})
	}
}

func TestFindBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test executable
	testBinary := "test_lmutil"
	if runtime.GOOS == "windows" {
		testBinary = "test_lmutil.exe"
	}
	testBinaryPath := filepath.Join(tmpDir, testBinary)
	f, err := os.Create(testBinaryPath)
	if err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}
	f.Close()

	// Make it executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(testBinaryPath, 0755); err != nil {
			t.Fatalf("Failed to chmod test binary: %v", err)
		}
	}

	// Test finding a binary that doesn't exist
	result := FindBinary("nonexistent_binary_xyz123")
	// Should return default path, not empty string
	if result == "" {
		t.Error("FindBinary should return a default path for non-existent binary, got empty string")
	}

	// Test that the function returns a path
	result = FindBinary("lmutil")
	if result == "" {
		t.Error("FindBinary returned empty string")
	}
}

func TestGetDefaultBinaryPaths(t *testing.T) {
	paths := GetDefaultBinaryPaths()

	expectedBinaries := []string{
		"lmutil",
		"rlmstat",
		"spmstat",
		"sesictrl",
		"rvlstatus",
		"tlm_server",
		"pixar_query",
	}

	for _, binary := range expectedBinaries {
		if path, exists := paths[binary]; !exists {
			t.Errorf("GetDefaultBinaryPaths() missing %q", binary)
		} else if path == "" {
			t.Errorf("GetDefaultBinaryPaths() returned empty path for %q", binary)
		}
	}
}
