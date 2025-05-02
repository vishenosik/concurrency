package concurrency

import (
	"log"
	"strconv"
	"testing"
	"time"
)

func Main_Scheduler(t *testing.T) {

	scheduler, err := NewScheduler()
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

		scheduler.Add(NewIntervalJob(
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

// MockQueueManager for testing
type MockQueueManager struct {
	nextFunc  func() (time.Time, bool)
	runFunc   func()
	addFunc   func(job Job)
	nextCalls int
	runCalls  int
	addCalls  int
}

func (m *MockQueueManager) Next() (time.Time, bool) {
	m.nextCalls++
	if m.nextFunc != nil {
		return m.nextFunc()
	}
	return time.Time{}, false
}

func (m *MockQueueManager) Run() {
	m.runCalls++
	if m.runFunc != nil {
		m.runFunc()
	}
}

func (m *MockQueueManager) Add(job Job) {
	m.addCalls++
	if m.addFunc != nil {
		m.addFunc(job)
	}
}

func TestNewScheduler(t *testing.T) {
	t.Run("default scheduler", func(t *testing.T) {
		s, err := NewScheduler()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if s.idleTimeout != time.Hour {
			t.Errorf("Expected idleTimeout of 1 hour, got %v", s.idleTimeout)
		}

		if s.queue == nil {
			t.Error("Expected queue to be initialized")
		}
	})

	t.Run("with options", func(t *testing.T) {
		mockQM := &MockQueueManager{}
		s, err := NewScheduler(WithQueueManager(mockQM))
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if s.queue != mockQM {
			t.Error("Expected custom queue manager to be set")
		}
	})
}

func TestMustInitScheduler(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("MustInitScheduler panicked unexpectedly")
		}
	}()

	s := MustInitScheduler()
	if s == nil {
		t.Error("Expected scheduler instance, got nil")
	}
}

func TestScheduler_Add(t *testing.T) {
	mockQM := &MockQueueManager{
		addFunc: func(job Job) {},
	}
	s := &Scheduler{
		queue:  mockQM,
		addCH:  make(chan struct{}, 1),
		stopCH: make(chan struct{}),
	}

	job := func() {}
	s.Add(NewIntervalJob("", time.Microsecond, time.Now(), job))

	if mockQM.addCalls != 1 {
		t.Errorf("Expected Add to be called once, got %d", mockQM.addCalls)
	}

	select {
	case <-s.addCH:
		// Success
	default:
		t.Error("Expected addCH to receive a signal")
	}
}

func TestScheduler_RunJob(t *testing.T) {
	runCalled := false
	mockQM := &MockQueueManager{
		runFunc: func() {
			runCalled = true
		},
	}
	s := &Scheduler{queue: mockQM}

	s.runJob()

	if !runCalled {
		t.Error("Expected Run to be called")
	}
}

func TestScheduler_RunJobWithPanic(t *testing.T) {
	mockQM := &MockQueueManager{
		runFunc: func() {
			panic("test panic")
		},
	}
	s := &Scheduler{queue: mockQM}

	// This should not panic
	s.runJob()
}

func TestWithQueueManager(t *testing.T) {
	mockQM := &MockQueueManager{}
	s := defaultScheduler()

	opt := WithQueueManager(mockQM)
	opt(s)

	if s.queue != mockQM {
		t.Error("WithQueueManager did not set the queue manager")
	}

	// Test nil case
	s = defaultScheduler()
	opt = WithQueueManager(nil)
	opt(s)

	if s.queue == nil {
		t.Error("WithQueueManager should not set nil queue manager")
	}
}

type FailingQueueManager struct{}

func (f *FailingQueueManager) Next() (time.Time, bool) {
	return time.Time{}, false
}

func (f *FailingQueueManager) Run() {
	panic("intentional panic for testing")
}

func (f *FailingQueueManager) Add(job Job) {}

func TestScheduler_RunJobPanic(t *testing.T) {
	s := &Scheduler{
		queue: &FailingQueueManager{},
	}

	// This should not panic
	s.runJob()
}

func TestScheduler_AddAfterStop(t *testing.T) {
	s := &Scheduler{
		queue:  &MockQueueManager{},
		addCH:  make(chan struct{}, 1),
		stopCH: make(chan struct{}),
	}

	s.Stop()

	// This should not panic
	s.Add(NewIntervalJob("", time.Microsecond, time.Now(), func() {}))
}
