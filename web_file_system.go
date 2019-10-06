package main

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// WebFileSystem provides some checks and restrictions around http.FileSystem during file serving.
// WebFileSystem prohibits:
// * paths with dots (eg. /some/path/../foo, /some/path/./bar)
// * files without names (eg. .env) - such files are often sensitive and should be hidden
// * directories.  (serves only files, request to path that is directory returns 404)
type WebFileSystem struct {
	http.Dir
}

// Open is a wrapper around the Open method of the embedded FileSystem and provides necessary checks
func (fs WebFileSystem) Open(requestedPath string) (http.File, error) {
	// Check requestedPath contain dots
	if pathContainsDots(requestedPath) {
		return nil, os.ErrNotExist
	}

	absolutePath := filepath.Join(string(fs.Dir), filepath.FromSlash(path.Clean("/"+requestedPath)))

	// Check requestedPath is directory
	fileInfo, fileInfoError := os.Stat(absolutePath)
	if nil != fileInfoError {
		return nil, os.ErrNotExist
	}
	if fileInfo.IsDir() {
		return nil, os.ErrNotExist
	}

	file, fileOpenError := fs.Dir.Open(requestedPath)
	if nil != fileOpenError {
		return nil, fileOpenError
	}

	return file, nil
}

// The path is assumed to be a delimited by forward slashes, as guaranteed by the http.FileSystem interface.
func pathContainsDots(path string) bool {
	for _, part := range strings.Split(path, "/") {
		if len(part) > 0 && "." == part[0:1] {
			return true
		}
	}

	return false
}

func NewWebFileSystem(rootDir string) *WebFileSystem {
	return &WebFileSystem{
		Dir: http.Dir(rootDir),
	}
}
