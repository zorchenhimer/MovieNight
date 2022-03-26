package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type FileSystem interface {
	fs.FS

	// WriteFiles writes all the files in the embedded filesystem to the disk
	//
	// WriteFiles should return the number of files written even if an error occured
	WriteFiles(name string) (int, error)
}

type ReadFileDirFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type ioFS struct {
	diskDir    string
	replaceKey string
	fsys       ReadFileDirFS
}

func (f ioFS) Open(name string) (fs.File, error) {
	if file, err := os.Open(f.diskPath(name)); err == nil {
		return file, nil
	}

	file, err := f.fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not find file on disk in %s or in embedded FS: %w", f.diskDir, err)
	}

	return file, nil
}

func (f ioFS) WriteFiles(name string) (int, error) {
	writeCount := 0

	entries, err := f.fsys.ReadDir(name)
	if err != nil {
		return 0, fmt.Errorf("could not read fs directory %s: %w", name, err)
	}

	for _, entry := range entries {
		entryName := path.Join(name, entry.Name())
		diskName := f.diskPath(entryName)
		if entry.IsDir() {
			if _, err := os.Open(diskName); errors.Is(err, os.ErrNotExist) {
				err = os.MkdirAll(diskName, 0644)
				if err != nil {
					return writeCount, fmt.Errorf("could not create directory %s: %w", diskName, err)
				}
			}

			count, err := f.WriteFiles(entryName)
			writeCount += count
			if err != nil {
				return writeCount, fmt.Errorf("could not write files for %s: %w", entryName, err)
			}
		} else {
			_, err := os.Open(diskName)
			if err == nil {
				continue
			}

			if errors.Is(err, os.ErrNotExist) {
				data, err := f.fsys.ReadFile(entryName)
				if err != nil {
					return writeCount, fmt.Errorf("could not read fs file %s: %w", entryName, err)
				}

				err = os.WriteFile(diskName, data, 0644)
				if err != nil {
					return writeCount, fmt.Errorf("could not write data from %s to %s: %w", entryName, diskName, err)
				}

				writeCount += 1
			}
		}
	}

	return writeCount, nil
}

func (f ioFS) diskPath(name string) string {
	return path.Join(f.diskDir, strings.TrimPrefix(name, f.replaceKey))
}

func FS(fsys ReadFileDirFS, diskDir, replaceKey string) (FileSystem, error) {
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
