//go:build dev
// +build dev

package common

import (
	"log"
	"os"
)

var logDev *log.Logger = log.New(os.Stdout, "[DEV]", log.LstdFlags)

func LogDevf(format string, v ...interface{}) {
	logDev.Printf(format, v...)
}

func LogDevln(v ...interface{}) {
	logDev.Println(v...)
}
