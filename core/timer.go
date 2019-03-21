package core

import (
	"container/heap"
	"sync"
	"time"
)

func NewTimerEngine() *TimerEngine {
	tick, err := time.ParseDuration(Config().Core.TimerTick)
	if err != nil {
		Fatal("failed to parse timer engine tick: %v", err)
	}
	return &TimerEngine{evPool{}, tick, Config().Core.TimerParallel}
}

type TimerEngine struct {
	pool     evPool
	tick     time.Duration
	parallel int
}

func (t *TimerEngine) Run(initEv ...Event) error {
	for _, ev := range initEv {
		t.pool.Push(ev)
	}
	permission := make(chan struct{}, t.parallel)
	var underway counter
	for {
		event, empty := t.pool.Pop()
		if empty && underway.Value() == 0 {
			break
		}
		if event != nil {
			underway.Add(1)
			go func() {
				defer underway.Sub(1)
				permission <- struct{}{}
				defer func() { <-permission }()
				if err := t.pool.Handle(event); err != nil {
					Debug("failed to handle %s: %v", event, Shorten(err.Error(), 100))
				}
			}()
		} else {
			time.Sleep(t.tick)
		}
	}
	return nil
}

type Event interface {
	Deadline() time.Time
	Trigger() (chain []Event, err error)
	String() string
}

type evPool struct {
	sync.Mutex
	evheap evMinHeap
}

func (s *evPool) Handle(event Event) error {
	chain, err := event.Trigger()
	if len(chain) > 0 {
		s.Push(chain...)
	}
	return err
}

func (s *evPool) Push(events ...Event) {
	s.Lock()
	defer s.Unlock()
	for _, event := range events {
		if event != nil {
			Debug("heap <+  %s", event)
			heap.Push(&s.evheap, event)
		}
	}
}

func (s *evPool) Pop() (event Event, empty bool) {
	s.Lock()
	defer s.Unlock()
	if len(s.evheap) == 0 {
		return nil, true
	}
	if time.Now().Before(s.evheap[0].Deadline()) {
		return nil, false
	}
	event = heap.Pop(&s.evheap).(Event)
	Debug("heap  -> %s", event)
	return event, false
}

type evMinHeap []Event

func (e *evMinHeap) Len() int {
	return len(*e)
}
func (e *evMinHeap) Less(i, j int) bool {
	return (*e)[i].Deadline().Before((*e)[j].Deadline())
}
func (e *evMinHeap) Swap(i, j int) {
	(*e)[i], (*e)[j] = (*e)[j], (*e)[i]
}
func (e *evMinHeap) Push(x interface{}) {
	*e = append(*e, x.(Event))
}
func (e *evMinHeap) Pop() interface{} {
	size := e.Len()
	if size <= 0 {
		return nil
	}
	last := (*e)[size-1]
	(*e)[size-1] = nil
	*e = (*e)[:size-1]
	return last
}

type counter struct {
	sync.Mutex
	N int
}

func (r *counter) Add(i int) {
	r.Lock()
	defer r.Unlock()
	r.N += i
}

func (r *counter) Sub(i int) {
	r.Lock()
	defer r.Unlock()
	r.N -= i
}
func (r *counter) Value() int {
	r.Lock()
	defer r.Unlock()
	return r.N
}
