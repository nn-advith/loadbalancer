package strategy

import (
	"fmt"
	"sync"
)

type Strategy interface {
	Initialise([]string) error
	Next() string
}

var StrategyMap = map[string]Strategy{
	"RoundRobin": &RoundRobin{},
}

// ROund RObin strategy
type RoundRobin struct {
	backends []string
	index    int
	lock     sync.Mutex
}

func (r *RoundRobin) Initialise(backends []string) error {
	if len(backends) == 0 {
		return fmt.Errorf("empty backends list")
	}
	r.backends = backends
	r.index = 0
	r.lock = sync.Mutex{}
	return nil
}

func (r *RoundRobin) Next() string {
	r.lock.Lock()
	backend := r.backends[r.index]
	r.index = (r.index + 1) % len(r.backends)
	r.lock.Unlock()
	return backend
}

// Weighted Round Robin strategy

// type WRoundRobin struct {
// 	backends []string
// 	weights []float64
// 	index int
// 	lock sync.Mutex
// }
