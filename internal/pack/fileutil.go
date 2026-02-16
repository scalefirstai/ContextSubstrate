package pack

import "os"

// openFileExclusive opens a file for writing, creating it if it doesn't exist.
// Returns an error if the file already exists.
func openFileExclusive(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
}
