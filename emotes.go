package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
	"golang.org/x/exp/slices"
)

var emotesDir string

func loadEmotes() error {
	var err error
	common.Emotes, err = processEmoteDir(emotesDir)
	if err != nil {
		return fmt.Errorf("could not process emote dir: %w", err)
	}
	return nil
}

func processEmoteDir(dir string) (common.EmotesMap, error) {
	em := make(common.EmotesMap)
	_, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			common.LogInfof("%s does not exist so no emotes were loaded\n", dir)
		} else {
			common.LogErrorf("could not open emote dir: %v\n", err)
		}
		return em, nil
	}

	filepath.WalkDir(dir, func(fpath string, d fs.DirEntry, err error) error {
		if d.IsDir() || err != nil {
			return nil
		}

		if slices.Contains([]string{".png", ".gif"}, filepath.Ext(fpath)) {
			em = em.Add(path.Join("emotes", strings.TrimPrefix(filepath.ToSlash(fpath), dir)))
		}

		return nil
	})

	common.LogInfof("Found %d emotes in %s\n", len(em), dir)

	return em, nil
}
