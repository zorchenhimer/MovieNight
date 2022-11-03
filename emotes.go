package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
	"github.com/zorchenhimer/MovieNight/files"
)

var emotesLocation string

func init() {
	emotesLocation = files.JoinRunPath("emotes")
}

func loadEmotes() error {
	var err error
	common.Emotes, err = processEmoteDir(emotesLocation)
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

	// Find top level emotes
	em, err = findEmotes(dir, em)
	if err != nil {
		return nil, fmt.Errorf("could not find emotes in top level directory: %w", err)
	}

	// Get second level subdirs (eg, "twitch", "zorchenhimer", etc)
	for _, subDir := range subDirs {
		subd, err := os.ReadDir(path.Join(dir, subDir))
		if err != nil {
			common.LogErrorf("Error reading dir %q: %v\n", subd, err)
			continue
		}
		for _, d := range subd {
			if d.IsDir() {
				p := path.Join(dir, subDir, d.Name())
				em, err = findEmotes(p, em)
				if err != nil {
					common.LogErrorf("Error finding emotes in %q: %v\n", p, err)
				}
			}
		}
	}

	common.LogInfof("processEmoteDir: %d\n", len(em))
	return em, nil
}

func findEmotes(dir string, em common.EmotesMap) (common.EmotesMap, error) {
	dir = filepath.ToSlash(dir)
	common.LogDebugf("finding emotes in %q\n", dir)

	for _, ext := range []string{"*.png", "*.gif"} {
		files, err := filepath.Glob(path.Join(dir, ext))
		if err != nil {
			return nil, fmt.Errorf("unable to glob emote directory with %q: %w", ext, err)
		}
		common.LogInfof("Found %d %s emotes\n", len(files), ext)

		for _, file := range files {
			em = em.Add(path.Join("emotes", strings.TrimPrefix(filepath.ToSlash(file), dir)))
		}
	}

	return em, nil
}
