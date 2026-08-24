package syncutil

import (
	"sync"
)

// FanOut spawns a new goroutine each time `Run()` is called, up to a max of `maxConcurrent`
// concurrent requests is reached. Subsequent calls to `Run()` will block until previously `Run()`
// routines have completed. `Wait()` then collects any errors from the routines once they
// have all completed.
type FanOut struct {
	errCh   chan error
	limiter chan struct{}
	errs    []error
	wg      sync.WaitGroup
}

func NewFanOut(maxConcurrent int) *FanOut {
	// They probably want no concurrency
	if maxConcurrent == 0 {
		maxConcurrent = 1
	}

	pool := FanOut{
		errCh:   make(chan error, maxConcurrent),
		limiter: make(chan struct{}, maxConcurrent),
		errs:    make([]error, 0),
	}

	pool.wg.Add(1)
	go pool.start()
	return &pool
}

func (p *FanOut) start() {
	defer p.wg.Done()
	for err := range p.errCh {
		p.errs = append(p.errs, err)
	}
}

// Run a new routine with an optional data value
func (p *FanOut) Run(callBack func(any) error, data any) {
	p.limiter <- struct{}{}
	go func() {
		err := callBack(data)
		if err != nil {
			p.errCh <- err
		}
		<-p.limiter
	}()
}

// Wait for all the routines to complete and return any errors.
// This MUST be called after N calls to Run()
func (p *FanOut) Wait() []error {
	for i := 0; i < cap(p.limiter); i++ {
		p.limiter <- struct{}{}
	}
	if p.errCh != nil {
		close(p.errCh)
	}
	p.wg.Wait()

	if len(p.errs) == 0 {
		return nil
	}
	return p.errs
}
