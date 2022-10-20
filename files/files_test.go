package files_test

import (
	"testing"

	"github.com/zorchenhimer/MovieNight/files"
)

func TestFSNil(t *testing.T) {
	_, err := files.FS(nil, "", "")
	if err == nil {
		t.Error("no error was returned when a nil fsys was passed in")
	}
}
