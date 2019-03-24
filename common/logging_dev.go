// +build dev

package common

func LogDevf(format string, v ...interface{}) {
	if logError == nil {
		panic("Logging not setup!")
	}

	logError.Printf(format, v...)
}

func LogDevln(v ...interface{}) {
	if logError == nil {
		panic("Logging not setup!")
	}

	logError.Println(v...)
}
