package util

import (
"context"
"fmt"
"sync"
"time"
)

// NewSchedule creates a task scheduler that limits a maximal number of tasks run concurrently
// using Gate (unlimited if nil), and will delay each task execution by provided amount.
//
// Only one task per token (function argument) will be run at once.
//
// If max is set, each task that gets rescheduled further than this amount will be forced to execution
// regardless of actual schedule.
func NewSchedule(delay, max time.Duration, gate *Gate, fnc func(interface{})) *Schedule {
	s := &Schedule{
		delay: delay, max: max, gate: gate, fnc: fnc,
		close:      make(chan struct{}),
		done:       make(chan struct{}),
		reschedule: make(chan struct{}, 1),
		running:    make(map[interface{}]<-chan struct{}),
	}
	go s.loop()
	return s
}

type schedTask struct {
	tok   interface{}
	targ  time.Time
	start time.Time
	done  chan struct{}
}

// Schedule schedules tasks.
type Schedule struct {
	delay time.Duration
	max   time.Duration
	gate  *Gate
	fnc   func(interface{})

	close chan struct{}
	done  chan struct{}

	reschedule chan struct{}

	mu        sync.Mutex
	wg        sync.WaitGroup
	scheduled []schedTask

	rmu     sync.Mutex
	running map[interface{}]<-chan struct{}
}

func (s *Schedule) markRunning(tok interface{}, done <-chan struct{}) {
	s.rmu.Lock()
	if _, ok := s.running[tok]; ok {
		s.rmu.Unlock()
		panic(fmt.Errorf("%v is already running", tok))
	}
	s.running[tok] = done
	s.rmu.Unlock()
}

func (s *Schedule) notRunning(tok interface{}) {
	s.rmu.Lock()
	if _, ok := s.running[tok]; !ok {
		s.rmu.Unlock()
		panic("tasks are running concurrently for token")
	}
	delete(s.running, tok)
	s.rmu.Unlock()
}

func (s *Schedule) isRunning(tok interface{}) bool {
	s.rmu.Lock()
	_, ok := s.running[tok]
	s.rmu.Unlock()
	return ok
}

func (s *Schedule) setTasks(run, add []schedTask) {
	for i, t := range run {
		if t.done == nil {
			t.done = make(chan struct{})
			run[i] = t
		}
	}
	for i, t := range add {
		if t.done == nil {
			t.done = make(chan struct{})
			add[i] = t
		}
	}
	s.mu.Lock()
	for _, t := range run {
		s.run(t)
	}
	s.scheduled = add
	s.mu.Unlock()
	select {
	case s.reschedule <- struct{}{}:
	default:
	}
}

func (s *Schedule) run(ct schedTask) {
	if s.gate != nil {
		if !s.gate.StartCh(s.close) {
			return
		}
	}
	s.markRunning(ct.tok, ct.done)
	s.wg.Add(1)
	go func(tok interface{}, done chan struct{}) {
		defer func() {
			if s.gate != nil {
				s.gate.Done()
			}
			s.wg.Done()
			if done != nil {
				close(done)
			}
			s.notRunning(tok)
		}()
		s.fnc(tok)
	}(ct.tok, ct.done)
}

func (s *Schedule) loop() {
	var (
		timer *time.Timer
	)
	scheduleNext := func() {
		if len(s.scheduled) == 0 {
			if timer != nil {
				timer.Stop()
			}
			timer = nil
			return
		}
		now := time.Now()
		dt := s.scheduled[0].targ.Sub(now)
		if s.max > 0 {
			for _, ct := range s.scheduled {
				if mt := ct.start.Add(s.max).Sub(now); mt < dt {
					dt = mt
				}
			}
		}
		if timer != nil {
			timer.Reset(dt)
		} else {
			timer = time.NewTimer(dt)
		}
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
		s.wg.Wait()
		close(s.done)
	}()
	for {
		var t <-chan time.Time
		if timer != nil {
			t = timer.C
		}
		select {
		case <-s.close:
			return
		case <-s.reschedule:
			s.mu.Lock()
			scheduleNext()
			s.mu.Unlock()
			continue
		case <-t:
			timer = nil
		}
		func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			now := time.Now()
			last := 0
			// first run all tasks with target time before current moment
			for i := 0; i < len(s.scheduled); i++ {
				ct := s.scheduled[i]
				if ct.targ.After(now) {
					last = i
					break
				}
				s.scheduled = append(s.scheduled[:i], s.scheduled[i+1:]...)
				i--
				if !s.isRunning(ct.tok) {
					s.run(ct)
				}
			}
			if s.max > 0 { // force execution of old tasks, if requested
				// TODO(dennwc): eliminate second scan
				for i := last; i < len(s.scheduled); i++ {
					ct := s.scheduled[i]
					if now.Sub(ct.start) > s.max && !s.isRunning(ct.tok) {
						s.scheduled = append(s.scheduled[:i], s.scheduled[i+1:]...)
						i--
						s.run(ct)
					}
				}
			}
			scheduleNext()
		}()
	}
}

// Close will cancel all scheduled tasks and wait for all running tasks to finish.
func (s *Schedule) Close() error {
	select {
	case <-s.close:
		return nil
	default:
	}
	close(s.close)
	<-s.done
	return nil
}

// Schedule adds task to a schedule and returns channel that closes when
// task completes. Provided token must be comparable.
//
// If task is scheduled again before it has been executed, it will be delayed
// further instead of executing task twice.
func (s *Schedule) Schedule(tok interface{}) <-chan struct{} {
	s.mu.Lock()
	start := time.Now()
	nt := start.Add(s.delay)
	reschedule := len(s.scheduled) == 0
	var done chan struct{}
	for i, ct := range s.scheduled {
		if ct.tok == tok {
			done = ct.done
			start = ct.start
			reschedule = reschedule || i == 0 || (s.max > 0 && start.Add(s.max).Before(s.scheduled[0].targ))
			s.scheduled = append(s.scheduled[:i], s.scheduled[i+1:]...)
			break
		}
	}
	if done == nil {
		done = make(chan struct{})
	}
	s.scheduled = append(s.scheduled, schedTask{
		tok: tok, targ: nt, done: done, start: start,
	})
	if reschedule {
		select {
		case s.reschedule <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
	return done
}

// Cancel removes task from schedule. If task was already running it will not be stopped automatically.
func (s *Schedule) Cancel(tok interface{}) {
	s.mu.Lock()
	for i, ct := range s.scheduled {
		if ct.tok == tok {
			if ct.tok != nil {
				select {
				case <-ct.done:
				default:
					close(ct.done)
				}
			}
			s.scheduled = append(s.scheduled[:i], s.scheduled[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
}

// Wait blocks until all running tasks will be finished, or context is canceled.
func (s *Schedule) Wait(ctx context.Context) error {
	for {
		var wait <-chan struct{}
		s.mu.Lock()
		if len(s.scheduled) != 0 {
			wait = s.scheduled[0].done
		}
		s.mu.Unlock()
		if wait == nil {
			s.rmu.Lock()
			for _, done := range s.running {
				if done != nil {
					wait = done
					break
				}
			}
			s.rmu.Unlock()
		}
		if wait == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wait:
		}
	}
}

