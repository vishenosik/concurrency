package concurrency

import (
	"log"
	"testing"
	"time"
)

func Test_Scheduler(t *testing.T) {
	scheduler := NewScheduler()
	scheduler.Start()
	defer scheduler.Stop()

	// Add 1000 tasks with different intervals
	for i := 1; i < 3; i++ {
		taskID := i
		interval := time.Second * time.Duration(i) // Varying intervals 1-60 seconds

		if i == 2 {
			time.Sleep(time.Second * 5)
		}

		log.Println(taskID, interval)

		scheduler.AddTask(&Job{
			ID:       taskID,
			Interval: interval,
			NextRun:  time.Now().Add(time.Duration(i) * time.Millisecond), // Stagger start times
			Job: func() {
				time.Sleep(time.Millisecond * 400)
				log.Printf("Task %d executed at %v\n", taskID, time.Now().Format("15:04:05.000"))
			},
		})
	}

	// Run for 5 minutes
	time.Sleep(5 * time.Minute)
}

func Test_Scheduler_v2(t *testing.T) {
	queue := NewHeapQueue()
	scheduler := NewScheduler_v2(queue)
	scheduler.Start()
	defer scheduler.Stop()

	// Add 1000 tasks with different intervals
	for i := 1; i < 3; i++ {
		taskID := i
		interval := time.Second * time.Duration(i) // Varying intervals 1-60 seconds

		if i == 2 {
			time.Sleep(time.Second * 5)
		}

		log.Println(taskID, interval)

		queue.AddJob(&Job{
			ID:       taskID,
			Interval: interval,
			NextRun:  time.Now().Add(time.Duration(i) * time.Millisecond), // Stagger start times
			Job: func() {
				time.Sleep(time.Millisecond * 400)
				log.Printf("Task %d executed at %v\n", taskID, time.Now().Format("15:04:05.000"))
			},
		})
	}

	// Run for 5 minutes
	time.Sleep(5 * time.Minute)
}
