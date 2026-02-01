package system

import (
	"time"
)

var StartTime time.Time

func InitStartTime() {
	StartTime = time.Now()
}

func Uptime() int64 {
	uptime := time.Since(StartTime)
	return int64(uptime.Seconds())
}
