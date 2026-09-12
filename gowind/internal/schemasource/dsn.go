package schemasource

import (
	"os"
	"strings"
)

// NormalizeDSN normalizes the DSN to ensure it has a valid scheme.
// If it already has a scheme (mysql://, postgres://, etc.), it's returned as-is.
// If it's a file path, it will be prefixed with "file://".
// Otherwise, it's treated as SQL text content and prefixed with "text://".
func NormalizeDSN(dsn string) string {
	// Check if it already has a scheme
	if strings.Contains(dsn, "://") {
		return dsn
	}

	// Check if it's a file path
	if _, err := os.Stat(dsn); err == nil {
		return "file://" + dsn
	}

	// Treat it as SQL text content
	return "text://" + dsn
}
