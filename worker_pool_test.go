package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewWorkerPool(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		pool := NewWorkerPool()
		require.Equal(t, int32(3), pool.minWorkers)
		require.Equal(t, int32(256), pool.maxWorkers)
		require.Equal(t, int32(10), pool.initWorkers)
	})

	t.Run("custom configuration", func(t *testing.T) {
		pool := NewWorkerPool(WithWorkersControl(5, 20, 10))
		require.Equal(t, int32(5), pool.minWorkers)
		require.Equal(t, int32(20), pool.maxWorkers)
		require.Equal(t, int32(10), pool.initWorkers)
	})

	t.Run("auto-adjust init workers", func(t *testing.T) {
		pool := NewWorkerPool(WithWorkersControl(5, 10, 3)) // init < min
		require.Equal(t, int32(5), pool.initWorkers)

		pool = NewWorkerPool(WithWorkersControl(5, 10, 15)) // init > max
		require.Equal(t, int32(10), pool.initWorkers)

		pool = NewWorkerPool(WithWorkersControl(10, 5, 15)) // min > max
		require.Equal(t, int32(5), pool.minWorkers)
	})
}

func Test_WorkerPool_Lifecycle(t *testing.T) {
	pool := NewWorkerPool(
		WithWorkersControl(2, 5, 2),
		WithTimeouts(time.Millisecond*100, time.Millisecond*150),
	)
	pool.Start(context.TODO())

	time.Sleep(time.Second)

	require.Equal(t, int32(2), pool.numWorkers.Load())

	// Test worker scaling
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = pool.AddTask(Task{Func: func() { time.Sleep(time.Millisecond * 200) }})
		}()
	}
	wg.Wait()

	// Wait for scale down
	time.Sleep(1 * time.Second)
	require.Equal(t, int32(2), pool.numWorkers.Load())

	pool.Stop(context.TODO())

	// Verify pool is closed
	_, err := pool.AddTask(Task{Func: func() {}})
	require.ErrorIs(t, err, ErrPoolClosed)
}

func Test_WorkerPool_TaskExecution(t *testing.T) {
	pool := NewWorkerPool(WithWorkersControl(1, 1, 1))
	pool.Start(context.TODO())
	defer pool.Stop(context.TODO())

	var executed atomic.Bool
	_, err := pool.AddTask(Task{Func: func() { executed.Store(true) }})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	require.True(t, executed.Load())
}

func Test_WorkerPool_ConcurrentTaskSubmission(t *testing.T) {
	pool := NewWorkerPool(WithWorkersControl(5, 20, 5))
	pool.Start(context.TODO())
	defer pool.Stop(context.TODO())

	var counter atomic.Int32
	const numTasks = 1000
	var wg sync.WaitGroup

	wg.Add(numTasks)
	for range numTasks {
		go func() {
			defer wg.Done()
			_, err := pool.AddTask(Task{Func: func() { counter.Add(1) }})
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
	time.Sleep(1000 * time.Millisecond) // Allow all tasks to complete

	require.Equal(t, int32(numTasks), counter.Load())
}

func Test_WorkerPool_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPoolContext(ctx, WithWorkersControl(1, 1, 1))
	pool.Start(context.TODO())

	cancel()
	time.Sleep(time.Millisecond * 50)

	_, err := pool.AddTask(Task{Func: func() {}})
	require.ErrorIs(t, err, ErrPoolClosed)
}

func Test_WorkerPool_PanicRecovery(t *testing.T) {
	pool := NewWorkerPool(WithWorkersControl(1, 1, 1))
	pool.Start(context.TODO())
	defer pool.Stop(context.TODO())

	var wg sync.WaitGroup
	wg.Add(1)

	_, err := pool.AddTask(Task{Func: func() {
		defer wg.Done()
		panic("test panic")
	}})
	require.NoError(t, err)

	wg.Wait()
}

func Test_WorkerPool_CloseBehavior(t *testing.T) {
	pool := NewWorkerPool(WithWorkersControl(1, 1, 1))
	pool.Start(context.TODO())

	// Add task before closing
	_, err := pool.AddTask(Task{Func: func() {}})
	require.NoError(t, err)

	pool.Stop(context.TODO())

	require.True(t, pool.closed.Load())

	// Verify channels are closed
	select {
	case _, ok := <-pool.addCH:
		require.False(t, ok)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for task channel")
	}
}

func BenchmarkWorkerPool(b *testing.B) {
	pool := NewWorkerPool(WithWorkersControl(10, 100, 10))
	pool.Start(context.TODO())
	defer pool.Stop(context.TODO())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = pool.AddTask(Task{Func: func() {
				// Simulate work
				time.Sleep(10 * time.Microsecond)
			}})
		}
	})
}

func BenchmarkWorkerPoolHeavyWork(b *testing.B) {
	pool := NewWorkerPool(WithWorkersControl(10, 100, 10))
	pool.Start(context.TODO())
	defer pool.Stop(context.TODO())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = pool.AddTask(Task{Func: func() {
				// Simulate heavier work
				time.Sleep(1 * time.Millisecond)
			}})
		}
	})
}

func BenchmarkWorkerPoolScaleUp(b *testing.B) {
	pool := NewWorkerPool(
		WithWorkersControl(1, 100, 1),
		WithTimeouts(10*time.Millisecond, 0),
	)
	pool.Start(context.TODO())
	defer pool.Stop(context.TODO())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pool.AddTask(Task{Func: func() {
			time.Sleep(100 * time.Microsecond)
		}})
	}
}
