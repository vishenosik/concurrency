package concurrency

import (
	"log"
	"time"
)

type IntervalJob struct {
	id       string
	interval time.Duration
	lastRun  time.Time
	nextRun  time.Time
	run      func()
}

func NewIntervalJob(id string, interval time.Duration, nextRun time.Time, job func()) *IntervalJob {
	return &IntervalJob{
		id:       id,
		interval: interval,
		nextRun:  nextRun,
		run:      job,
	}
}

func (jb *IntervalJob) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job %s panicked: %v\n", jb.id, r)
		}
	}()
	jb.run()
}

func (jb *IntervalJob) Reset() {
	now := time.Now()
	jb.lastRun = now
	jb.nextRun = now.Add(jb.interval)
}

func (jb *IntervalJob) NextRun() time.Time {
	return jb.nextRun
}
