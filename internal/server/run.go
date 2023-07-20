package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Tap-Team/timerticker/internal/config"
	"github.com/Tap-Team/timerticker/internal/domain/timerticker"
	"github.com/Tap-Team/timerticker/internal/transport/grpcserver"
	"github.com/Tap-Team/timerticker/internal/transport/grpcserver/timerservice"
)

func Run() {
	config := config.FromFile("config/config.yaml")

	ticker := timerticker.New()
	defer ticker.Close()
	go func() {
		ticker.Start(context.Background(), time.Second)
	}()

	timerService := timerservice.New(ticker)

	server := grpcserver.New(
		&grpcserver.Services{
			TimerService: timerService,
		},
	)

	log.Fatal(server.ListenAndServe(fmt.Sprintf("%s:%d", config.ServerConfig.Host, config.ServerConfig.Port)))
}
