package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrdjeb/trueproxy/internal/config"
	"github.com/mrdjeb/trueproxy/internal/logger"
	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/proxy"
)

func main() {
	cfg := config.MustLoad()

	log := logger.Set(cfg.LogEnviroment)

	log.Info(
		"starting trueproxy",
		slog.String("env", cfg.LogEnviroment),
		slog.String("addr", cfg.Address),
	)
	log.Debug("debug messages are enabled")

	cm, err := proxy.NewCertManager(cfg.Cert)
	if err != nil {
		log.Error("Faild init cert manager", sl.Err(err))
		os.Exit(1)
	}

	srv := http.Server{
		Handler:           proxy.New(log, cm),
		Addr:              cfg.Address,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler)), //disable http2
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("listen and serve returned err: ", sl.Err(err))
		}
	}()

	log.Debug("server started")
	sig := <-quit
	log.Debug("handle quit chanel: ", slog.Any("os.Signal", sig.String()))
	log.Debug("server stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShotdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown returned an err: ", sl.Err(err))
	}

	log.Debug("server stopped")

}
