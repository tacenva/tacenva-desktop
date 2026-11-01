package discovery

import "sync"

type State struct {
	mu sync.RWMutex

	done  bool
	err   error
	count int

	callbacks []func()
}

func NewState() *State {
	return &State{}
}

func (s *State) Done() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.done
}

func (s *State) Error() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.err
}

func (s *State) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.count
}

func (s *State) OnDone(callback func()) {
	s.mu.Lock()

	if s.done {
		s.mu.Unlock()

		callback()
		return
	}

	s.callbacks = append(
		s.callbacks,
		callback,
	)

	s.mu.Unlock()
}

func (s *State) Complete(
	count int,
	err error,
) {
	s.mu.Lock()

	s.done = true
	s.count = count
	s.err = err

	callbacks := append(
		[]func(){},
		s.callbacks...,
	)

	s.callbacks = nil

	s.mu.Unlock()

	for _, callback := range callbacks {
		callback()
	}
}
