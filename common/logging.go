package common

import (
	"fmt"
	"io"
	"log"
	"os"
)

type LogLevel string

const (
	LLError LogLevel = "error" // only log errors
	LLChat  LogLevel = "chat"  // log chat and commands
	LLInfo  LogLevel = "info"  // log info messages (not quite debug, but not chat)
	LLDebug LogLevel = "debug" // log everything
)

const (
	logPrefixError string = "[ERROR] "
	logPrefixChat  string = "[CHAT] "
	logPrefixInfo  string = "[INFO] "
	logPrefixDebug string = "[DEBUG] "
)

var (
	logError *log.Logger
	logChat  *log.Logger
	logInfo  *log.Logger
	logDebug *log.Logger
)

func SetupLogging(level LogLevel, file string) error {
	switch level {
	case LLDebug:
		if file == "" {
			logError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
			logChat = log.New(os.Stdout, logPrefixChat, log.LstdFlags)
			logDebug = log.New(os.Stdout, logPrefixDebug, log.LstdFlags)
			logInfo = log.New(os.Stdout, logPrefixInfo, log.LstdFlags)
		} else {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("unable to open log file for writing: %w", err)
			}
			logError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
			logChat = log.New(io.MultiWriter(os.Stdout, f), logPrefixChat, log.LstdFlags)
			logInfo = log.New(io.MultiWriter(os.Stdout, f), logPrefixInfo, log.LstdFlags)
			logDebug = log.New(io.MultiWriter(os.Stdout, f), logPrefixDebug, log.LstdFlags)
		}
	case LLChat:
		logDebug = nil
		if file == "" {
			logError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
			logChat = log.New(os.Stdout, logPrefixChat, log.LstdFlags)
			logInfo = log.New(os.Stdout, logPrefixInfo, log.LstdFlags)
		} else {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("unable to open log file for writing: %w", err)
			}
			logError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
			logChat = log.New(io.MultiWriter(os.Stdout, f), logPrefixChat, log.LstdFlags)
			logInfo = log.New(io.MultiWriter(os.Stdout, f), logPrefixInfo, log.LstdFlags)
		}

	case LLInfo:
		logDebug = nil
		logChat = nil
		if file == "" {
			logError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
			logInfo = log.New(os.Stdout, logPrefixInfo, log.LstdFlags)
		} else {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("unable to open log file for writing: %w", err)
			}
			logError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
			logInfo = log.New(io.MultiWriter(os.Stdout, f), logPrefixInfo, log.LstdFlags)
		}

	// Default to error
	default:
		logChat = nil
		logDebug = nil
		logInfo = nil
		if file == "" {
			logError = log.New(os.Stderr, logPrefixError, log.LstdFlags)
		} else {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("unable to open log file for writing: %w", err)
			}
			logError = log.New(io.MultiWriter(os.Stderr, f), logPrefixError, log.LstdFlags)
		}
	}
	return nil
}

func LogErrorf(format string, v ...interface{}) {
	if logError == nil {
		panic("Logging not setup!")
	}

	logError.Printf(format, v...)
}

func LogErrorln(v ...interface{}) {
	if logError == nil {
		panic("Logging not setup!")
	}

	logError.Println(v...)
}

func LogChatf(format string, v ...interface{}) {
	// if logError isn't set to something, logging wasn't setup.
	if logError == nil {
		panic("Logging not setup!")
	}

	// logging chat and commands is turned off.
	if logChat == nil {
		return
	}

	logChat.Printf(format, v...)
}

func LogChatln(v ...interface{}) {
	// if logError isn't set to something, logging wasn't setup.
	if logError == nil {
		panic("Logging not setup!")
	}

	// logging chat and commands is turned off.
	if logChat == nil {
		return
	}

	logChat.Println(v...)
}

func LogInfof(format string, v ...interface{}) {
	// if logError isn't set to something, logging wasn't setup.
	if logError == nil {
		panic("Logging not setup!")
	}

	// logging info is turned off.
	if logInfo == nil {
		return
	}

	logInfo.Printf(format, v...)
}

func LogInfoln(v ...interface{}) {
	// if logError isn't set to something, logging wasn't setup.
	if logError == nil {
		panic("Logging not setup!")
	}

	// logging info is turned off.
	if logInfo == nil {
		return
	}

	logInfo.Println(v...)
}

func LogDebugf(format string, v ...interface{}) {
	// if logError isn't set to something, logging wasn't setup.
	if logError == nil {
		panic("Logging not setup!")
	}

	// logging debug is turned off.
	if logDebug == nil {
		return
	}

	logDebug.Printf(format, v...)
}

func LogDebugln(v ...interface{}) {
	// if logError isn't set to something, logging wasn't setup.
	if logError == nil {
		panic("Logging not setup!")
	}

	// logging debug is turned off.
	if logDebug == nil {
		return
	}

	logDebug.Println(v...)
}
