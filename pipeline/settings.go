package pipeline

func (pln *pipeline[Type]) SetSteps(steps ...handlerFunc[Type]) *pipeline[Type] {
	pln.steps = steps
	return pln
}

func (pln *pipeline[Type]) SetCommit(commit func(Type)) *pipeline[Type] {
	pln.commit = commit
	return pln
}

func (pln *pipeline[Type]) SetGoroutinesLimit(limit int) *pipeline[Type] {
	if limit > 0 {
		pln.goroutinesLimit = limit
	}
	return pln
}

func (pln *pipeline[Type]) DisableStepPanicRecover() *pipeline[Type] {
	pln.recoverPanics = false
	return pln
}

func (pln *pipeline[Type]) SetRecoveryHandler(handler Recoverer) *pipeline[Type] {
	if handler != nil {
		pln.recoverer = handler
	}
	return pln
}
