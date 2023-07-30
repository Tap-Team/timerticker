package timerservice

import (
	"context"
	"errors"

	"github.com/Tap-Team/timerticker/internal/errorutils/timererror"
	"github.com/Tap-Team/timerticker/proto/timerservicepb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var empty emptypb.Empty

type Ticker interface {
	UpdateTime(timerId uuid.UUID, newEndTime int64) error
	AddTimer(timerId uuid.UUID, endTime int64) error
	AddManyTimers(timersEndTime map[int64][]uuid.UUID) error
	RemoveTimer(timerId uuid.UUID) error
	NewStream() interface {
		Stream() <-chan []uuid.UUID
		Close()
	}
}

type Service struct {
	ticker Ticker
	timerservicepb.UnimplementedTimerServiceServer
}

func New(
	ticker Ticker,
) *Service {
	return &Service{
		ticker: ticker,
	}
}

func Error(err error) error {
	switch {
	case errors.Is(err, timererror.ErrTimerAlreadyExpired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, timererror.ErrTimerNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, timererror.ErrTimerExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Unknown, err.Error())
	}
}

func (s *Service) Add(ctx context.Context, event *timerservicepb.AddEvent) (*emptypb.Empty, error) {
	err := s.ticker.AddTimer(uuid.UUID(event.GetTimerId()), event.GetEndTime())
	if err != nil {
		return &empty, Error(err)
	}
	return &empty, nil
}
func (s *Service) AddMany(ctx context.Context, event *timerservicepb.AddManyEvent) (*emptypb.Empty, error) {
	timers := make(map[int64][]uuid.UUID)
	for _, event := range event.GetEvents() {
		timers[event.GetEndTime()] = append(timers[event.GetEndTime()], uuid.UUID(event.GetTimerId()))
	}
	err := s.ticker.AddManyTimers(timers)
	if err != nil {
		return &empty, Error(err)
	}
	return &empty, nil
}
func (s *Service) Start(ctx context.Context, event *timerservicepb.StartEvent) (*emptypb.Empty, error) {
	err := s.ticker.AddTimer(uuid.UUID(event.GetTimerId()), event.GetEndTime())
	if err != nil {
		return &empty, Error(err)
	}
	return &empty, nil
}
func (s *Service) Stop(ctx context.Context, event *timerservicepb.StopEvent) (*emptypb.Empty, error) {
	err := s.ticker.RemoveTimer(uuid.UUID(event.GetTimerId()))
	if err != nil {
		return &empty, Error(err)
	}
	return &empty, nil
}
func (s *Service) Remove(ctx context.Context, event *timerservicepb.RemoveEvent) (*emptypb.Empty, error) {
	err := s.ticker.RemoveTimer(uuid.UUID(event.GetTimerId()))
	if err != nil {
		return &empty, Error(err)
	}
	return &empty, nil
}
func (s *Service) Update(ctx context.Context, event *timerservicepb.UpdateEvent) (*emptypb.Empty, error) {
	err := s.ticker.UpdateTime(uuid.UUID(event.GetTimerId()), event.GetEndTime())
	if err != nil {
		return &empty, Error(err)
	}
	return &empty, nil
}

func (s *Service) TimerTick(e *emptypb.Empty, stream timerservicepb.TimerService_TimerTickServer) error {
	str := s.ticker.NewStream()
	defer str.Close()
	for timerIds := range str.Stream() {
		ids := make([][]byte, 0, len(timerIds))
		for i := range timerIds {
			ids = append(ids, timerIds[i][:])
		}
		stream.Send(&timerservicepb.TimerFinishEvent{Ids: ids})
	}
	return nil
}
