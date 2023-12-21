package util

import (
	"fmt"
	"log"

	"github.com/getsentry/sentry-go"
)

func logToSentry(msg string) {
	if IsSentryOn() {
		// bad looking error in Sentry if use out of box:
		// - not right message
		// - redundant top stack frame - this one
		// sentry.CaptureException(fmt.Errorf(format, args...))

		stacktrace := sentry.NewStacktrace()
		if l := len(stacktrace.Frames); l > 0 {
			stacktrace.Frames = stacktrace.Frames[:l-1]
		}

		event := sentry.NewEvent()
		event.Level = sentry.LevelError
		event.Message = msg
		event.Exception = []sentry.Exception{{
			Value: msg,
			// Type:       reflect.TypeOf(exception).String(),
			Stacktrace: stacktrace,
		}}

		hub := sentry.CurrentHub()
		client := hub.Client()
		client.CaptureEvent(event, nil, &sentryScope)
	}
}

func Error(args ...interface{}) {
	log.Println(args...)
	logToSentry(fmt.Sprint(args...))
}

func Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
	logToSentry(fmt.Sprintf(format, args...))
}

func Info(args ...interface{}) {
	log.Println(args...)
}

func Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

var DebugFlag bool

func Debug(args ...interface{}) {
	if !DebugFlag {
		return
	}
	log.Println(args...)
}

func Debugf(format string, args ...interface{}) {
	if !DebugFlag {
		return
	}
	log.Printf(format, args...)
}
