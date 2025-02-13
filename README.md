# Concurrency package

This is a Golang library to implement popular concurrency patterns in a painless for developers.

# Documentation

[CONTRIBUTING](./docs/CHANGELOG.md)

[CHANGELOG](./docs/CHANGELOG.md)

[RELEASING](./docs/RELEASING.md)

# Usage

Learn more about usage in testing files.

```go
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
```