package util

import (
	"os"
	"path/filepath"
	"runtime"
)

// FindBinary searches for a binary in multiple locations with OS-specific handling
// On Windows, it automatically adds .exe extension and checks current directory
// On Unix/Linux, it checks standard paths like /usr/local/bin
func FindBinary(baseName string) string {
	// Determine the binary name based on OS
	binaryName := baseName
	if runtime.GOOS == "windows" {
		binaryName = baseName + ".exe"
	}

	// Search locations in order of preference
	searchPaths := []string{
		// 1. Current executable directory (useful for portable installations on Windows)
		filepath.Join(getExecutableDir(), binaryName),
		// 2. Current working directory
		binaryName,
		// 3. Standard Unix/Linux path
		filepath.Join("/usr/local/bin", baseName),
	}

	// On Windows, also check without .exe in standard path (for flexibility)
	if runtime.GOOS == "windows" {
		searchPaths = append(searchPaths, filepath.Join("/usr/local/bin", baseName))
	}

	// Try each path
	for _, path := range searchPaths {
		if fileExists(path) {
			return path
		}
	}

	// If nothing found, return the standard Unix path as fallback
	// This maintains backward compatibility
	if runtime.GOOS == "windows" {
		// On Windows, prefer current directory as fallback
		return binaryName
	}
	return filepath.Join("/usr/local/bin", baseName)
}

// getExecutableDir returns the directory containing the current executable
func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// fileExists checks if a file exists and is accessible
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetDefaultBinaryPaths returns a map of default binary paths for all supported license tools
// This uses cross-platform detection to find binaries in appropriate locations
func GetDefaultBinaryPaths() map[string]string {
	return map[string]string{
		"lmutil":      FindBinary("lmutil"),
		"rlmstat":     FindBinary("rlmutil"),
		"spmstat":     FindBinary("spmstat"),
		"sesictrl":    FindBinary("sesictrl"),
		"rvlstatus":   FindBinary("rvlstatus"),
		"tlm_server":  FindBinary("tlm_server"),
		"pixar_query": FindBinary("pixar_query.sh"),
	}
}
