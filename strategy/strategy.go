package strategy

import (
	"fmt"
	"math/rand"
	"slices"
	"sync"

	"github.com/nn-advith/loadbalancer/utils/common"
	parser "github.com/nn-advith/loadbalancer/utils/parser"
)

// helper functions

type Strategy interface {
	Initialise([]parser.Backend) error
	Next() string
}

var StrategyMap = map[string]Strategy{
	"RoundRobin":         &RoundRobin{},
	"WeightedRoundRobin": &WeightedRoundRobin{},
}

// ROund RObin strategy
type RoundRobin struct {
	backends []string
	index    int
	lock     sync.Mutex
}

func (r *RoundRobin) Initialise(backends []parser.Backend) error {
	bl := parser.ConstructBackendURIs(backends)
	if len(bl) == 0 {
		return fmt.Errorf("empty backends list")
	}
	r.backends = bl
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

// Weighted Round Robin strategy (SWRR)

type WeightedRoundRobin struct {
	backends   []string
	weights    []float64
	index      int
	subtractor float64
	lock       sync.Mutex
}

func GetSubtractor(nums []float64) float64 {
	return 0.1
}

func (w *WeightedRoundRobin) Initialise(backends []parser.Backend) error {
	bl := parser.ConstructBackendURIs(backends)
	bw := parser.GetWeights(backends)

	if len(bl) == 0 {
		return fmt.Errorf("empty backends list")
	}
	if len(bw) == 0 {
		return fmt.Errorf("empty weight list")
	}

	w.backends = bl
	w.weights = bw
	w.lock = sync.Mutex{}
	w.index = slices.Index(bw, slices.Max(bw))
	w.subtractor = GetSubtractor(w.weights)

	return nil
}

func (w *WeightedRoundRobin) Next() string {
	w.lock.Lock()
	backend := w.backends[w.index]

	//update index to next
	w.weights[w.index] = w.weights[w.index] - w.subtractor

	nextvals := common.IndexAll(w.weights, slices.Max(w.weights))
	w.index = nextvals[rand.Intn(len(nextvals))]

	w.lock.Unlock()
	return backend
}
