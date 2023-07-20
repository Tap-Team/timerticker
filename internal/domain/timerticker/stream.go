package timerticker

import "github.com/google/uuid"

type Stream interface {
	Stream() <-chan []uuid.UUID
	Close()
}

type CompletedTimerStream struct {
	id     uuid.UUID
	ticker *Ticker
	stream <-chan []uuid.UUID
}

func (s *CompletedTimerStream) Stream() <-chan []uuid.UUID {
	return s.stream
}

func (s *CompletedTimerStream) Close() {
	s.ticker.subscribers.Unsubscribe(s.id)
}

func (t *Ticker) NewStream() interface {
	Stream() <-chan []uuid.UUID
	Close()
} {
	id := uuid.New()
	ch := t.subscribers.Subscribe(id)
	return &CompletedTimerStream{id: id, stream: ch, ticker: t}
}
