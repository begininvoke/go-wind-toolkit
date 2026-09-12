package project

import (
	"os"
	"path/filepath"
	"strings"
)

// getGoModProjectRoot gets the root directory of the Go module project.
func getGoModProjectRoot(dir string) string {
	if dir == filepath.Dir(dir) {
		return dir
	}
	if goModIsNotExistIn(dir) {
		return getGoModProjectRoot(filepath.Dir(dir))
	}
	return dir
}

// goModIsNotExistIn checks if go.mod file does not exist in the specified directory.
func goModIsNotExistIn(dir string) bool {
	_, e := os.Stat(filepath.Join(dir, "go.mod"))
	return os.IsNotExist(e)
}

// isDirExists checks if the directory exists.
func isDirExists(baseDir, projectName string) bool {
	to := filepath.Join(baseDir, projectName)
	if _, err := os.Stat(to); !os.IsNotExist(err) {
		return true
	}
	return false
}

// processProjectParams process project name and working dir.
//
// "~/name" expands to the user home directory; a bare "~" or "~name" is
// treated as a literal relative path like any other. The result is resolved
// to an absolute path and split into (base, dir) = (project name, parent dir).
func processProjectParams(projectName string) (projectNameResult, workingDirResult string) {
	dir := projectName
	if strings.HasPrefix(projectName, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(homeDir, strings.TrimPrefix(projectName, "~/"))
		}
	}
	if !filepath.IsAbs(dir) {
		if absPath, err := filepath.Abs(dir); err == nil {
			dir = absPath
		}
	}
	return filepath.Base(dir), filepath.Dir(dir)
}
