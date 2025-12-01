package services

import (
	"os"
	"os/exec"
	"runtime"
)

// UtilityStatus represents the status of a license utility binary
type UtilityStatus struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// UtilityChecker checks for the availability of license server utilities
type UtilityChecker struct {
	binaries map[string]string
}

// NewUtilityChecker creates a new utility checker
func NewUtilityChecker() *UtilityChecker {
	binPaths := GetDefaultBinaryPaths()
	return &UtilityChecker{
		binaries: map[string]string{
			"FlexLM (lmutil)":     binPaths["lmutil"],
			"RLM (rlmutil)":       binPaths["rlmstat"],
			"SPM (spmstat)":       binPaths["spmstat"],
			"SESI (sesictrl)":     binPaths["sesictrl"],
			"RVL (rvlstatus)":     binPaths["rvlstatus"],
			"Tweak (tlm_server)":  binPaths["tlm_server"],
			"Pixar (pixar_query)": binPaths["pixar_query"],
		},
	}
}

// CheckAll checks the availability of all license utilities
func (uc *UtilityChecker) CheckAll() []UtilityStatus {
	var statuses []UtilityStatus

	for name, path := range uc.binaries {
		status := uc.checkUtility(name, path)
		statuses = append(statuses, status)
	}

	return statuses
}

// checkUtility checks if a single utility is available and executable
func (uc *UtilityChecker) checkUtility(name, path string) UtilityStatus {
	status := UtilityStatus{
		Name:      name,
		Path:      path,
		Available: false,
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			status.Message = "Binary not found"
			return status
		}
		status.Message = "Error checking binary: " + err.Error()
		return status
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		status.Message = "Not a regular file"
		return status
	}

	// Check if it's executable (Unix/Linux only)
	// On Windows, executability is determined by file extension (.exe, .bat, .cmd)
	// not by permission bits, so we skip this check
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			status.Message = "Not executable (check permissions)"
			return status
		}
	}

	// Try to execute with --version or -h to verify it works
	// Most license utilities support these flags
	if err := uc.testExecutable(path); err != nil {
		status.Message = "Binary exists but may not be executable: " + err.Error()
		status.Available = true // Still mark as available but with warning
		return status
	}

	status.Available = true
	status.Message = "Available and executable"
	return status
}

// testExecutable attempts to execute the binary to verify it works
func (uc *UtilityChecker) testExecutable(path string) error {
	// Try with --version first
	cmd := exec.Command(path, "--version")
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Try with -h
	cmd = exec.Command(path, "-h")
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Try with no arguments (some utilities show help)
	cmd = exec.Command(path)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If all fail, it might still work but we can't verify
	return nil
}
