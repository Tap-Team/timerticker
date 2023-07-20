package grpcserver

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/Tap-Team/timerticker/proto/timerservicepb"
	"github.com/go-kit/log"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Services struct {
	TimerService timerservicepb.TimerServiceServer
}

type Server struct {
	services *Services
	*grpc.Server
}

func New(
	services *Services,
) *Server {

	// Define customfunc to handle panic
	customFunc := func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	}
	// Shared options for the logger, with a custom gRPC code to log level function.
	opts := []recovery.Option{
		recovery.WithRecoveryHandler(customFunc),
	}

	logger := log.NewLogfmtLogger(os.Stdout)
	rpcLogger := log.With(logger, "service", "gRPC/server", "component", "timer-ticker")
	logTraceID := func(ctx context.Context) logging.Fields {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return logging.Fields{"traceID", span.TraceID().String()}
		}
		return nil
	}

	return &Server{
		services: services,
		// server with recovery and logging
		Server: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				recovery.UnaryServerInterceptor(opts...),
				logging.UnaryServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
			),
			grpc.ChainStreamInterceptor(
				recovery.StreamServerInterceptor(opts...),
				logging.StreamServerInterceptor(interceptorLogger(rpcLogger), logging.WithFieldsFromContext(logTraceID)),
			),
		),
	}
}

func (s *Server) RegisterServices() {
	timerservicepb.RegisterTimerServiceServer(s, s.services.TimerService)
}

func (s *Server) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Join(errors.New("failed to listen server"), err)
	}
	err = s.Serve(lis)
	if err != nil {
		return errors.Join(errors.New("failed to serve listener"), err)
	}
	return nil
}

func (s *Server) Serve(lis net.Listener) error {
	var err error
	s.RegisterServices()
	err = s.Server.Serve(lis)
	if err != nil {
		return errors.Join(errors.New("failed to serve listener"), err)
	}
	return nil
}
