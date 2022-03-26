package files

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ioFS struct {
	diskDir    string
	replaceKey string
	fsys       fs.FS
}

func (f ioFS) Open(name string) (fs.File, error) {
	diskPath := path.Join(f.diskDir, strings.TrimPrefix(name, f.replaceKey))
	if file, err := os.Open(diskPath); err == nil {
		return file, nil
	}

	file, err := f.fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not find file on disk in %q or in embedded FS: %w", f.diskDir, err)
	}

	return file, nil
}

func FS(fsys fs.FS, diskDir, replaceKey string) (fs.FS, error) {
	if fsys == nil {
		return nil, fmt.Errorf("fsys is null")
	}

	if diskDir == "" {
		// replaceKey is cleared because there is no diskDir to replace with the base dir with
		replaceKey = ""
		diskDir = RunPath()
	}

	return ioFS{
		diskDir:    diskDir,
		replaceKey: replaceKey,
		fsys:       fsys,
	}, nil
}

// Return the absolut directory containing the MovieNight binary
func RunPath() string {
	ex, er := os.Executable()
	if er != nil {
		panic(er)
	}
	dir := filepath.ToSlash(filepath.Dir(ex))
	return strings.TrimPrefix(dir, filepath.VolumeName(dir))
}
