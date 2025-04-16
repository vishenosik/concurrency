package concurrency

type Recoverer interface {
	Recover(message any)
}

type noopRecoveryHandler struct{}

func (h *noopRecoveryHandler) Recover(_ any) {}

func (pln *pipeline[Type]) _recover() {
	if !pln.recoverPanics {
		return
	}
	if r := recover(); r != nil {
		pln.recoverer.Recover(r)
	}
}
