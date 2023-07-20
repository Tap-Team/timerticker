package timerticker

import (
	"context"
	"sync"
	"time"

	"github.com/Tap-Team/timerticker/internal/errorutils/timererror"
	"github.com/google/uuid"
)

type subscribers struct {
	*sync.Mutex
	storage map[uuid.UUID]chan []uuid.UUID
}

func (s *subscribers) Subscribe(id uuid.UUID) chan []uuid.UUID {
	s.Lock()
	ch := make(chan []uuid.UUID, 100)
	s.storage[id] = ch
	s.Unlock()
	return ch
}

func (s *subscribers) Unsubscribe(id uuid.UUID) {
	s.Lock()
	ch, ok := s.storage[id]
	if !ok {
		return
	}
	delete(s.storage, id)
	close(ch)
	s.Unlock()
}

func (s *subscribers) SendAll(uuids []uuid.UUID) {
	s.Lock()
	for _, ch := range s.storage {
		ch <- uuids
	}
	s.Unlock()
}

func (s *subscribers) Close() {
	s.Lock()
	for u, ch := range s.storage {
		delete(s.storage, u)
		close(ch)
	}
	s.Unlock()
}

type Ticker struct {
	mu          sync.Mutex
	timers      map[uuid.UUID]int64
	endTime     map[int64][]uuid.UUID
	subscribers *subscribers
	cancel      func()
}

func New() *Ticker {
	return &Ticker{
		subscribers: &subscribers{Mutex: new(sync.Mutex), storage: make(map[uuid.UUID]chan []uuid.UUID)},
		timers:      make(map[uuid.UUID]int64),
		endTime:     make(map[int64][]uuid.UUID),
	}
}

func (t *Ticker) AddManyTimers(timersEndTime map[int64][]uuid.UUID) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().Unix()

	// validate endtime of timers
	for endTime := range timersEndTime {
		if now >= endTime {
			return timererror.ErrTimerAlreadyExpired
		}
	}
	for endTime, timerIds := range timersEndTime {
		t.endTime[endTime] = append(t.endTime[endTime], timerIds...)
		for _, id := range timerIds {
			t.timers[id] = endTime
		}
	}
	return nil
}

func (t *Ticker) AddTimer(timerId uuid.UUID, endTime int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().Unix()
	if now >= endTime {
		return timererror.ErrTimerAlreadyExpired
	}
	if _, ok := t.timers[timerId]; ok {
		return timererror.ErrTimerExists
	}
	t.timers[timerId] = endTime
	t.endTime[endTime] = append(t.endTime[endTime], timerId)
	return nil
}

func (t *Ticker) RemoveTimer(timerId uuid.UUID) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	// get timer end time and check timer is exists
	endTime, ok := t.timers[timerId]
	if !ok {
		return timererror.ErrTimerNotFound
	}
	// delete timer id from timers
	delete(t.timers, timerId)

	// delete timer from it endTime
	timers := t.endTime[endTime]
	for index, id := range timers {
		if id == timerId {
			timers[index] = timers[len(timers)-1]
			t.endTime[endTime] = timers[:len(timers)-1]
			return nil
		}
	}
	// recover to prev state
	t.timers[timerId] = endTime
	return timererror.ErrTimerNotFound
}

func (t *Ticker) Start(ctx context.Context, ticktime time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	for {
		time.Sleep(ticktime)
		select {
		case <-ctx.Done():
			// close chan
			return
		default:
			// launch go function
			go func() {
				// get current time
				now := time.Now().Unix()
				t.mu.Lock()
				// get timers which expired now
				timers, ok := t.endTime[now]
				// if ok send it to stream and delete from storage
				if ok {
					t.subscribers.SendAll(timers)
					for _, timerId := range timers {
						delete(t.timers, timerId)
					}
					delete(t.endTime, now)
				}
				t.mu.Unlock()
			}()
		}
	}
}

func (t *Ticker) Close() error {
	t.cancel()
	for k := range t.endTime {
		delete(t.endTime, k)
	}
	for u := range t.timers {
		delete(t.timers, u)
	}
	t.subscribers.Close()
	return nil
}
