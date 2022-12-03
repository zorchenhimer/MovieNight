package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
	"github.com/zorchenhimer/MovieNight/files"
	"golang.org/x/exp/slices"
)

func loadEmotes() error {
	var err error
	common.Emotes, err = processEmoteDir(files.JoinRunPath("emotes"))
	if err != nil {
		return fmt.Errorf("could not process emote dir: %w", err)
	}
	return nil
}

func processEmoteDir(dir string) (common.EmotesMap, error) {
	em := make(common.EmotesMap)
	dirInfo, err := os.ReadDir(dir)
	if err != nil {
		common.LogErrorf("could not open emote dir: %v\n", err)
		return em, nil
	}

	subDirs := []string{}

	for _, item := range dirInfo {
		// Get first level subdirs (eg, "twitch", "discord", etc)
		if item.IsDir() {
			subDirs = append(subDirs, item.Name())
			continue
		}
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
