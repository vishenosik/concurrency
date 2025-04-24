package concurrency

import (
	"log"
	"time"
)

type IntervalJob struct {
	id       string
	interval time.Duration
	last     time.Time
	next     time.Time
	run      func()
}

func NewIntervalJob(id string, interval time.Duration, Next time.Time, job func()) *IntervalJob {
	return &IntervalJob{
		id:       id,
		interval: interval,
		next:     Next,
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
	jb.last = now
	jb.next = now.Add(jb.interval)
}

func (jb *IntervalJob) Next() time.Time {
	return jb.next
}
