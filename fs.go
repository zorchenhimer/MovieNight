package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/zorchenhimer/MovieNight/common"
)

//go:embed static/*.html static/css static/img static/js
var staticFs embed.FS

func writeStaticFiles(name string) error {
	items, err := staticFs.ReadDir(name)
	if err != nil {
		return fmt.Errorf("could not read staticFs directory %#v: %w", name, err)
	}

	for _, item := range items {
		path := path.Join(name, item.Name())

		_, err := os.Open(path)
		notExist := errors.Is(err, os.ErrNotExist)

		if item.IsDir() {
			if notExist {
				common.LogInfof("creating dir %q\n", path)

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
			common.LogInfof("creating file %q\n", path)

			file, err := staticFs.Open(path)
			if err != nil {
				return fmt.Errorf("could not open embeded file %q: %w", path, err)
			}

			var staticData []byte
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

func setupStaticFiles() error {
	if err := writeStaticFiles("."); err != nil {
		return err
	}

	const staticEmotesDir = "static/emotes"
	if _, err := os.Open(staticEmotesDir); errors.Is(err, os.ErrNotExist) {
		common.LogInfof("creating dir %q\n", staticEmotesDir)
		if err := os.MkdirAll(staticEmotesDir, os.ModeDir); err != nil {
			return fmt.Errorf("could not create emotes directory: %w", err)
		}
	}

	return nil
}
