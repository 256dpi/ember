package ember

import (
	"io/fs"
	"strings"
)

// MustFiles will call Files and panic on errors.
func MustFiles(f fs.FS, dir string) map[string]string {
	// get files
	files, err := Files(f, dir)
	if err != nil {
		panic(err)
	}

	return files
}

// Files will return a file map from the provided file system directory.
func Files(f fs.FS, dir string) (map[string]string, error) {
	// trim dir
	dir = strings.Trim(dir, "/")

	// collect files
	files := make(map[string]string)
	err := fs.WalkDir(f, dir, func(path string, d fs.DirEntry, err error) error {
		// check error
		if err != nil {
			return err
		}

		// skip directories
		if d.IsDir() {
			return nil
		}

		// read file
		buf, err := fs.ReadFile(f, path)
		if err != nil {
			return err
		}

		// add file
		files[strings.TrimPrefix(path, dir+"/")] = string(buf)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
