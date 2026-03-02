package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	tsv1 "github.com/ummuys/pacttelegramservice/api/pb/v1"
	"github.com/ummuys/pacttelegramservice/pkg/config"
	"github.com/ummuys/pacttelegramservice/pkg/logger"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/adapter"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/repository"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/service"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/tgapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// TS -> TELEGRAM_SERVICE

type errMsg struct {
	err  error
	from string
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal(err)
	}

	logs, err := logger.InitLogger("report", "LOG_LEVEL_TELEGRAM_SERVICE")
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.ParseTelegramServiceConfig()
	if err != nil {
		logs.Fatal().Err(err).Msg("cfg")
	}

	lis, err := net.Listen(cfg.Network, cfg.Port)
	if err != nil {
		logs.Fatal().Err(err).Msg("listener")
	}

	// for tgapi //

	sr := repository.NewSessionRepository()
	sm := tgapi.NewSessionManager(ctx, cfg.AppID, cfg.AppHash, logs)
	if err != nil {
		logs.Fatal().Err(err).Str("component", "session_manager").Msg("")
	}
	ts := service.NewTelegramService(sm, sr, logs)

	tsAdpt := adapter.NewTelegramServiceAdapter(ts, logs)

	srv := grpc.NewServer()
	tsv1.RegisterTelegramServiceServer(srv, tsAdpt)
	reflection.Register(srv)

	wg := sync.WaitGroup{}
	errCh := make(chan errMsg, 1)

	wg.Go(func() {
		<-ctx.Done()
		srv.GracefulStop()
	})

	wg.Go(func() {
		logs.Info().Msg("Telegram service is running")
		if err := srv.Serve(lis); err != nil {
			errCh <- errMsg{
				err:  err,
				from: "grpc-server",
			}
		}
	})
	wg.Wait()
	close(errCh)

	haveErrs := false
	for msg := range errCh {
		if msg.err != nil {
			haveErrs = true
			logs.Error().Err(msg.err).Str("from", msg.from).Msg("")
		}
	}

	if haveErrs {
		logs.Info().Msg("Shutdown completed with errors")
	} else {
		logs.Info().Msg("Graceful shutdown completed")
	}

}
