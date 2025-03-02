package common

import "time"

type Interval struct {
	Start time.Duration
	End time.Duration
}

type CondenseFile struct {
	Input string
	Output string
	Sub string
	CondenseIntervals []*Interval
	OriginalDuration time.Duration
	CondensedDuration time.Duration
}
