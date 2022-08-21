package benchmark

import (
	"context"
	"sync"
)

type LimitWaitGroup struct {
	limit      uint
	occupiedCh chan struct{}
	wg         sync.WaitGroup
}

func NewLimitWaitGroup(limit uint) *LimitWaitGroup {
	return &LimitWaitGroup{
		limit:      limit,
		occupiedCh: make(chan struct{}, limit),
		wg:         sync.WaitGroup{},
	}
}

func (s *LimitWaitGroup) Add() {
	s.occupiedCh <- struct{}{}
	s.wg.Add(1)
}

func (s *LimitWaitGroup) AddWithContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.occupiedCh <- struct{}{}:
		s.wg.Add(1)
	}
	return nil
}

func (s *LimitWaitGroup) Done() {
	<-s.occupiedCh
	s.wg.Done()
}

func (s *LimitWaitGroup) Wait() {
	s.wg.Wait()
}
