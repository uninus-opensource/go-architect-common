package scheduler

import "time"

//IntervalType is interval duration type
type IntervalType int32

const (
	Year   IntervalType = 0
	Month  IntervalType = 1
	Day    IntervalType = 2
	Hour   IntervalType = 3
	Minute IntervalType = 4
	Second IntervalType = 5
)

//Job is job task to be run by scheduler
type Job struct {
	Name         string
	Command      string
	Repeat       int32
	Interval     int32
	IntervalType IntervalType
	StartTime    time.Time
	EndTime      time.Time
}

//Scheduler is task scheduler
type Scheduler interface {
	// AddOnceJob(...Job) error
	AddScheduledJob(Job) error
	DeleteJob(string) error
	// CancelJob(string) error
}
