package util

import (
	"time"

	"github.com/getsentry/sentry-go"
)

// SetupSentry turns on Sentry logging
func SetupSentry(dsn string) error {
	transport := sentry.NewHTTPTransport()
	// default is 30 seconds
	transport.Timeout = time.Second * 3

	return sentry.Init(sentry.ClientOptions{
		Dsn:       dsn,
		Transport: transport,
	})
}

type SentryScope struct {
	AppName string
}

var sentryScope SentryScope = SentryScope{}

// SetupSentryApp sets application name to Sentry events (cement != concrete, for example)
func SetupSentryApp(appName string) {
	sentryScope.AppName = appName
}

func (ss *SentryScope) ApplyToEvent(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	event.Logger = ss.AppName
	return event
}

func IsSentryOn() bool {
	return sentry.CurrentHub().Client() != nil
}
