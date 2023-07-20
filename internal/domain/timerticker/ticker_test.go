package timerticker_test

import (
	"context"
	"testing"
	"time"

	"github.com/Tap-Team/timerticker/internal/domain/timerticker"
	"github.com/Tap-Team/timerticker/internal/errorutils/timererror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAddRemove(t *testing.T) {
	var err error
	ticker := timerticker.New()
	timerUUID := uuid.New()
	// add timer expect no error
	err = ticker.AddTimer(timerUUID, time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)

	// try add timer with same uuid, expect error
	err = ticker.AddTimer(timerUUID, time.Now().Add(time.Hour*2).Unix())
	require.ErrorIs(t, err, timererror.ErrTimerExists)

	err = ticker.RemoveTimer(timerUUID)
	require.NoError(t, err)

	err = ticker.RemoveTimer(timerUUID)
	require.ErrorIs(t, err, timererror.ErrTimerNotFound)

	err = ticker.AddTimer(timerUUID, 0)
	require.ErrorIs(t, err, timererror.ErrTimerAlreadyExpired)
}

func TestStream(t *testing.T) {
	var err error

	timer1 := uuid.New()
	timer2 := uuid.New()
	timer3 := uuid.New()
	ticker := timerticker.New()
	endTime := time.Now().Add(time.Second).Unix()
	err = ticker.AddTimer(timer1, endTime)
	require.NoError(t, err, "add timer 1")
	err = ticker.AddTimer(timer2, endTime)
	require.NoError(t, err, "add timer 2")
	err = ticker.AddTimer(timer3, endTime)
	require.NoError(t, err, "add timer 3")

	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(endTime, 0).Add(time.Millisecond*500))
	defer cancel()
	go func() { ticker.Start(ctx, time.Second) }()

	err = ticker.RemoveTimer(timer3)
	require.NoError(t, err, "remove timer 3")
	for ids := range ticker.NewStream().Stream() {
		require.Equal(t, len(ids), 2)
		require.Equal(t, ids[0], timer1)
		require.Equal(t, ids[1], timer2)
	}
	ticker.Close()
}
