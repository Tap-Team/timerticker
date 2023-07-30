package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Tap-Team/timerticker/internal/config"
	"github.com/Tap-Team/timerticker/internal/domain/timerticker"
	"github.com/Tap-Team/timerticker/internal/transport/grpcserver"
	"github.com/Tap-Team/timerticker/internal/transport/grpcserver/timerservice"
)

func Run() {
	os.Setenv("TZ", "UTC")

	config := config.FromFile("config/config.yaml")

	ticker := timerticker.New()
	defer ticker.Close()
	go ticker.Start(context.Background(), time.Second)

	timerService := timerservice.New(ticker)

	server := grpcserver.New(
		&grpcserver.Services{
			TimerService: timerService,
		},
	)

	log.Fatal(server.ListenAndServe(fmt.Sprintf("%s:%d", config.ServerConfig.Host, config.ServerConfig.Port)))
}
