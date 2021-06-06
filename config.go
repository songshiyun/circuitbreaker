package circuitbreaker

import "time"

type Config struct {
	FailRate              uint8
	SlowCallRate          uint8
	WindowSize            uint32
	HalfOpenCalls         uint32
	MinNumOfCalls         uint32
	SlowCallDuration      time.Duration
	MaxDurationInHalfOpen time.Duration
	DurationInOpen        time.Duration
}


