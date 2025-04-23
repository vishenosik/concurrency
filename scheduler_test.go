package concurrency

import (
	"log"
	"strconv"
	"testing"
	"time"
)

func Test_Scheduler(t *testing.T) {

	queue, err := NewHeapQueue(NewJobList())
	if err != nil {
		panic(err)
	}

	scheduler, err := NewScheduler(queue)
	if err != nil {
		panic(err)
	}
	scheduler.Start()
	defer scheduler.Stop()

	// Add 1000 tasks with different intervals
	for i := 1; i < 3; i++ {
		taskID := strconv.Itoa(i)
		interval := time.Second * time.Duration(i) // Varying intervals 1-60 seconds

		if i == 2 {
			time.Sleep(time.Second * 5)
		}

		log.Println(taskID, interval)

		queue.AddJob(NewIntervalJob(
			taskID,
			interval,
			time.Now().Add(time.Duration(i)*time.Millisecond),
			func() {
				time.Sleep(time.Millisecond * 400)
				log.Printf("Task %s executed at %v\n", taskID, time.Now().Format("15:04:05.000"))
			},
		))
	}
}
