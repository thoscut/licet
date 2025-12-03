package services

import (
	"licet/internal/util"
)

// GetDefaultBinaryPaths returns a map of default binary paths for all supported license tools
// This uses cross-platform detection to find binaries in appropriate locations
func GetDefaultBinaryPaths() map[string]string {
	return util.GetDefaultBinaryPaths()
}
