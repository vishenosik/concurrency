package pipeline

import (
	"context"
	"fmt"
	"iter"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type Transaction struct {
	ID     int
	Amount int
}

type trans struct {
	data []Transaction
	mu   sync.Mutex
}

type Recovery struct{}

func (r *Recovery) Recover(msg any) { fmt.Println("Recovered in func", msg) }

func Test_Success(t *testing.T) {
	tr := NewTrans()

	NewPipeline(tr).
		SetSteps(filter, convert).
		SetCommit(tr.Commit).
		SetGoroutinesLimit(3).
		SetRecoveryHandler(new(Recovery)).
		Run()
}

func NewTrans() *trans {
	const size = 10

	t := &trans{
		data: make([]Transaction, 0, size),
	}

	for i := 0; i < size; i++ {
		t.data = append(t.data, Transaction{
			ID:     i,
			Amount: rand.Intn(500) - 100,
		})
	}

	return t

}

func (t *trans) Fetch(_ context.Context) iter.Seq[Transaction] {
	const size = 10

	return func(yield func(Transaction) bool) {
		for i := 0; i < size; i++ {
			if i == 9 {
				panic("ah man")
			}
			time.Sleep(time.Millisecond * 500)
			tr := Transaction{
				ID:     i,
				Amount: rand.Intn(500) - 100,
			}
			fmt.Printf("Transaction %d created...\n", i)

			if !yield(tr) {
				return
			}
		}
	}
}

func (t *trans) Fetch_(_ context.Context) iter.Seq[Transaction] {
	const size = 10
	res := make(chan Transaction, size)

	wg := sync.WaitGroup{}
	wg.Add(size)

	go func() {
		wg.Wait()
		close(res)
	}()

	for i := 0; i < size; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Second * 3)
			res <- Transaction{
				ID:     i,
				Amount: rand.Intn(500) - 100,
			}
			fmt.Printf("Transaction %d created...\n", i)
		}()
	}

	return func(yield func(Transaction) bool) {
		for r := range res {
			if !yield(r) {
				return
			}
		}
	}
}

func (t *trans) Fetch_2(ctx context.Context) iter.Seq[Transaction] {
	const size = 100
	res := make(chan Transaction, size)

	wg := sync.WaitGroup{}
	wg.Add(size)

	go func() {
		wg.Wait()
		close(res)
	}()

	for i := 0; i < size; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Second * 3)
			res <- Transaction{
				ID:     i,
				Amount: rand.Intn(500) - 100,
			}
			fmt.Printf("Transaction %d created...\n", i)
		}()
	}

	return func(yield func(Transaction) bool) {
		for r := range res {
			select {
			case <-ctx.Done():
				log.Println("FETCH ", ctx.Err())
				return
			default:
				fmt.Printf("Transaction %d read...\n", r.ID)
				if !yield(r) {
					return
				}
			}
		}
	}
}

func (t *trans) Commit(res Transaction) {
	time.Sleep(time.Second)
	// panic("trans")
	fmt.Printf("Transaction %d done (amount: %d)\n", res.ID, res.Amount)

}

func filter(tr Transaction) (Transaction, bool) {
	time.Sleep(time.Second * 3)
	fmt.Printf("Transaction %d filtered...(amount: %d)\n", tr.ID, tr.Amount)
	if tr.Amount >= 0 {
		return tr, true
	}
	return Transaction{}, false
}

func convert(tr Transaction) (Transaction, bool) {
	time.Sleep(time.Second)
	tr.Amount *= 2
	fmt.Printf("Transaction %d converted...(amount: %d)\n", tr.ID, tr.Amount)
	return tr, true
}
