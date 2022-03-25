package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
)

func loadEmotes() error {
	newEmotes, err := processEmoteDir(path.Join(common.RunPath(), "emotes"))
	if err != nil {
		return err
	}

	common.Emotes = newEmotes

	return nil
}

func processEmoteDir(path string) (common.EmotesMap, error) {
	dirInfo, err := os.ReadDir(path)
	if err != nil {
		common.LogErrorf("could not open emoteDir: %v\n", err)
	}

	subDirs := []string{}

	for _, item := range dirInfo {
		// Get first level subdirs (eg, "twitch", "discord", etc)
		if item.IsDir() {
			subDirs = append(subDirs, item.Name())
			continue
		}
	}

	em := common.NewEmotesMap()
	// Find top level emotes
	em, err = findEmotes(path, em)
	if err != nil {
		return nil, fmt.Errorf("could not findEmotes() in top level directory: %w", err)
	}

	// Get second level subdirs (eg, "twitch", "zorchenhimer", etc)
	for _, dir := range subDirs {
		subd, err := os.ReadDir(filepath.Join(path, dir))
		if err != nil {
			common.LogErrorf("Error reading dir %q: %v\n", subd, err)
			continue
		}
		for _, d := range subd {
			if d.IsDir() {
				p := filepath.Join(path, dir, d.Name())
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
	var runPathLength = len(common.RunPath() + "/emotes/")

	common.LogDebugf("finding emotes in %q\n", dir)
	emotePNGs, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		return em, fmt.Errorf("unable to glob emote directory: %w", err)
	}
	common.LogInfof("Found %d emotePNGs\n", len(emotePNGs))

	emoteGIFs, err := filepath.Glob(filepath.Join(dir, "*.gif"))
	if err != nil {
		return em, fmt.Errorf("unable to glob emote directory: %w", err)
	}
	common.LogInfof("Found %d emoteGIFs\n", len(emoteGIFs))

	for _, file := range emotePNGs {
		png := strings.ReplaceAll(common.Substr(file, runPathLength, len(file)), "\\", "/")
		//common.LogDebugf("Emote PNG: %s", png)
		em = em.Add(png)
	}

	for _, file := range emoteGIFs {
		gif := strings.ReplaceAll(common.Substr(file, runPathLength, len(file)), "\\", "/")
		//common.LogDebugf("Emote GIF: %s", gif)
		em = em.Add(gif)
	}

	return em, nil
}
