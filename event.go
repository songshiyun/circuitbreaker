package circuitbreaker

import "time"

type Event struct {
	Time    time.Time
	OldStat uint8
	NewStat uint8
}

type EventListenerFunc func(event *Event)
