package timerservice_test

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Tap-Team/timerticker/internal/domain/timerticker"
	"github.com/Tap-Team/timerticker/internal/transport/grpcserver"
	"github.com/Tap-Team/timerticker/internal/transport/grpcserver/timerservice"
	"github.com/Tap-Team/timerticker/proto/timerservicepb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	client timerservicepb.TimerServiceClient
)

const tickTime = time.Second

func TestMain(m *testing.M) {
	ctx := context.Background()
	ticker := timerticker.New()
	go func() { ticker.Start(ctx, tickTime) }()
	defer ticker.Close()

	timerService := timerservice.New(ticker)

	lis := bufconn.Listen(1 << 20)
	s := grpcserver.New(
		&grpcserver.Services{
			TimerService: timerService,
		},
	)
	go func() {
		err := s.Serve(lis)
		if err != nil {
			log.Fatalf("serve listener failed, %s", err)
		}
	}()

	dialer := func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial context failed, %s", err)
	}
	defer conn.Close()
	client = timerservicepb.NewTimerServiceClient(conn)
	m.Run()
}

func TestAdd(t *testing.T) {
	var err error
	ctx := context.Background()
	timerId := uuid.New()
	endTime := time.Now().Add(tickTime * 2).Unix()

	// simple add timer
	_, err = client.Add(ctx, &timerservicepb.AddEvent{TimerId: timerId[:], EndTime: endTime})
	require.NoError(t, err, "add timer failed")

	_, err = client.Add(ctx, &timerservicepb.AddEvent{TimerId: timerId[:], EndTime: endTime})
	s, ok := status.FromError(err)
	if !ok {
		t.Fatalf("add same timer error conversion failed, actual type is %T", err)
	}
	require.Equal(t, codes.AlreadyExists, s.Code(), "add same timer actual error code wrong, expected %s, actual %s", codes.AlreadyExists, s.Code())
	t.Cleanup(func() {
		client.Remove(ctx, &timerservicepb.RemoveEvent{TimerId: timerId[:]})
	})
}

func TestRemove(t *testing.T) {
	var err error
	ctx := context.Background()
	timerId := uuid.New()
	endTime := time.Now().Add(time.Second * 2).Unix()

	// simple add timer
	_, err = client.Add(ctx, &timerservicepb.AddEvent{TimerId: timerId[:], EndTime: endTime})
	require.NoError(t, err, "add timer failed")

	// remove timer, expect no error
	_, err = client.Remove(ctx, &timerservicepb.RemoveEvent{TimerId: timerId[:]})
	require.NoError(t, err, "remove timer failed")

	// try remove non-existent timer, assert error with code not found
	_, err = client.Remove(ctx, &timerservicepb.RemoveEvent{TimerId: timerId[:]})
	s, ok := status.FromError(err)
	if !ok {
		t.Fatalf("add same timer error conversion failed, actual type is %T", err)
	}
	require.Equal(t, codes.NotFound, s.Code(), "remove timer, actual error code wrong, expected %s, actual %s", codes.AlreadyExists, s.Code())
}

func TestStream(t *testing.T) {
	var err error
	ctx := context.Background()
	tNum := 10
	addEvents := make([]*timerservicepb.AddEvent, 0)
	now := time.Now().Add(tickTime * 2)
	for i := 0; i < tNum; i++ {
		uid := uuid.New()
		addEvents = append(addEvents, &timerservicepb.AddEvent{TimerId: uid[:], EndTime: now.Unix()})
		now = now.Add(tickTime)
	}
	_, err = client.AddMany(ctx, &timerservicepb.AddManyEvent{Events: addEvents})
	require.NoError(t, err, "add many failed")

	streamAmount := 3
	wg := new(sync.WaitGroup)
	wg.Add(streamAmount)

	for i := 0; i < streamAmount; i++ {
		go func() {
			i := 0
			stream := TickStream(t, ctx)
		Loop:
			for {
				select {
				case uuids, ok := <-stream:
					if !ok {
						break Loop
					}
					require.Equal(t, 1, len(uuids), "length of expired timers more than 1")
					i++
					t.Logf("%d timer recieved", i)
				case <-time.After(tickTime * 2):
					if i != 10 {
						log.Fatal("wrong count timers recieved")
					}
					break Loop
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

}

func TickStream(t *testing.T, ctx context.Context) <-chan []uuid.UUID {
	ctx, cancel := context.WithCancel(ctx)
	uuidChan := make(chan []uuid.UUID)
	var err error

	// get tick stream
	tick, err := client.TimerTick(ctx, &emptypb.Empty{})
	require.NoError(t, err, "listen timer tick failed")
	go func() {
		// in loop receive values
		for {
			event, err := tick.Recv()
			// if err is io.EOF break Loop
			if errors.Is(err, io.EOF) {
				t.Log("stream end, io.EOF")
				cancel()
				break
			}
			if err != nil {
				cancel()
				log.Fatalf("recieve event failed, %s", err)
			}
			// make list of uuid to send it to chan
			uuids := make([]uuid.UUID, 0, len(event.GetIds()))
			for _, b := range event.GetIds() {
				uuids = append(uuids, uuid.UUID(b))
			}
			uuidChan <- uuids
		}
		close(uuidChan)
	}()
	return uuidChan
}
