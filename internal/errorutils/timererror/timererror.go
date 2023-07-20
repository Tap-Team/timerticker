package timererror

import "errors"

var (
	ErrTimerNotFound       = errors.New("timer not found")
	ErrTimerExists         = errors.New("timer exists")
	ErrTimerAlreadyExpired = errors.New("endtime of time is expired")
)
