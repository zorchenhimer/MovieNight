package files

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

//go:embed settings_example.json
var settingsExampleFS embed.FS

var (
	DefaultFS FileSystem
)

type FileSystem struct {
	staticFS *embed.FS
	emotesFS *embed.FS
}

func (f FileSystem) openFS(name, fsName string, fs embed.FS) (fs.File, error) {
	file, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return fs.Open(fsName)
	}
	return file, nil
}

func (f FileSystem) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, "settings.json") {
		return f.openFS(name, "settings_example.json", settingsExampleFS)
	}

	if strings.HasPrefix(name, "emotes/") {
		return f.openFS(name, name, *f.emotesFS)
	}

	return f.openFS(name, name, *f.staticFS)
}

func (f FileSystem) readDirFS(name string, fs embed.FS) ([]fs.DirEntry, error) {
	dirs, err := os.ReadDir(name)
	if err != nil {
		return fs.ReadDir(name)
	}
	return dirs, nil
}

func (f FileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	if strings.HasPrefix(name, "emotes/") {
		return f.readDirFS(name, *f.emotesFS)
	}

	return f.readDirFS(name, *f.staticFS)
}

func (f FileSystem) ReadFile(name string) ([]byte, error) {
	file, err := f.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not get file size: %w", err)
	}

	data := make([]byte, fileInfo.Size())
	_, err = file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("could not read file data: %w", err)
	}
	return data, nil
}

func (f *FileSystem) RegisterStaticFS(fs *embed.FS) error {
	if f.staticFS != nil {
		return fmt.Errorf("static filesystem is already registered")
	}
	f.staticFS = fs
	return nil
}

func writeStaticFiles(name string) error {
	items, err := DefaultFS.ReadDir(name)
	if err != nil {
		return fmt.Errorf("could not read staticFs directory %#v: %w", name, err)
	}

	for _, item := range items {
		path := path.Join(name, item.Name())

		_, err := os.Open(path)
		notExist := errors.Is(err, os.ErrNotExist)

		if item.IsDir() {
			if notExist {
				fmt.Printf("creating dir %q\n", path)

				err = os.MkdirAll(path, os.ModeDir)
				if err != nil {
					return fmt.Errorf("could not make missing directory: %w", err)
				}
			}

			err = writeStaticFiles(path)
			if err != nil {
				return err
			}
		} else if notExist {
			fmt.Printf("creating file %q\n", path)

			file, err := DefaultFS.Open(path)
			if err != nil {
				return fmt.Errorf("could not open embeded file %q: %w", path, err)
			}

			fileInfo, err := file.Stat()
			if err != nil {
				return fmt.Errorf("could not get file size: %w", err)
			}

			staticData := make([]byte, fileInfo.Size())
			_, err = file.Read(staticData)
			if err != nil {
				return fmt.Errorf("could not read embeded file %q: %w", path, err)
			}

			err = os.WriteFile(path, staticData, 0644)
			if err != nil {
				return fmt.Errorf("could not write static data to file %q: %w", path, err)
			}
		}
	}
	return nil
}

func SetupStaticFiles() error {
	createDir := func(dir string) error {
		if _, err := os.Open(dir); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("creating dir %q\n", dir)
			if err := os.MkdirAll(dir, os.ModeDir); err != nil {
				return fmt.Errorf("could not create %q directory: %w", dir, err)
			}
		}
		return nil
	}

	if err := createDir("static"); err != nil {
		return err
	}

	if err := createDir("static/emotes"); err != nil {
		return err
	}

	if err := writeStaticFiles("static"); err != nil {
		return err
	}

	return nil
}
