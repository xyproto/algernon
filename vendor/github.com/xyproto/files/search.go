package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FindFile searches for files that contain the given substring in their filename
// starting from the current directory. It returns all matches as absolute paths
// or an error if the search fails.
func FindFile(substring string) ([]string, error) {
	var matches []string
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, errors.New("failed to get current directory: " + err.Error())
	}
	lower := strings.ToLower(substring)
	if err := filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.Contains(strings.ToLower(info.Name()), lower) {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}
			matches = append(matches, absPath)
		}
		return nil
	}); err != nil {
		return nil, errors.New("no matches when searching for " + substring)
	}
	return matches, nil
}

// HeaderSearch will search for a corresponding file, given a slice of extensions.
// This is useful for ie. finding a corresponding .h file for a .c file.
// The search starts in the current directory, then searches every parent directory in depth.
// TODO: Search sibling and parent directories named "include" first, then search the rest.
func HeaderSearch(absCppFilename string, headerExtensions []string, maxTime time.Duration) (string, error) {
	cppBasename := filepath.Base(absCppFilename)
	searchPath := filepath.Dir(absCppFilename)
	ext := filepath.Ext(cppBasename)
	if ext == "" {
		return "", errors.New("filename has no extension: " + cppBasename)
	}
	firstName := cppBasename[:len(cppBasename)-len(ext)]
	// First search the same path as the given filename, without using Walk
	withoutExt := strings.TrimSuffix(absCppFilename, ext)
	for _, hext := range headerExtensions {
		if Exists(withoutExt + hext) {
			return withoutExt + hext, nil
		}
	}
	headerNames := make([]string, 0, len(headerExtensions))
	for _, ext := range headerExtensions {
		headerNames = append(headerNames, firstName+ext)
	}
	foundHeaderAbsPath := ""
	startTime := time.Now()
	for {
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			basename := filepath.Base(info.Name())
			if err == nil {
				// logf("Walking %s\n", path)
				for _, headerName := range headerNames {
					if time.Since(startTime) > maxTime {
						return errors.New("file search timeout")
					}
					if basename == headerName {
						// Found the corresponding header!
						absFilename, err := filepath.Abs(path)
						if err != nil {
							continue
						}
						foundHeaderAbsPath = absFilename
						// logf("Found %s!\n", absFilename)
						return nil
					}
				}
			}
			// No result
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("error when searching for a corresponding header for %s: %w", cppBasename, err)
		}
		if len(foundHeaderAbsPath) == 0 {
			// Try the parent directory
			searchPath = filepath.Dir(searchPath)
			if len(searchPath) > 2 {
				continue
			}
		}
		break
	}
	if len(foundHeaderAbsPath) == 0 {
		return "", errors.New("found no corresponding header for " + cppBasename)
	}
	// Return the result
	return foundHeaderAbsPath, nil
}
