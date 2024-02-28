package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/mrdjeb/trueproxy/internal/api/handlers/request/list"
	"github.com/mrdjeb/trueproxy/internal/config"
	"github.com/mrdjeb/trueproxy/internal/logger"
	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/models"
	"github.com/mrdjeb/trueproxy/internal/proxy"
	"github.com/mrdjeb/trueproxy/internal/storage"
)

func main() {
	cfg := config.MustLoad()

	log := logger.Set(cfg.LogEnviroment)

	log.Info(
		"starting trueproxy",
		slog.String("env", cfg.LogEnviroment),
		slog.String("proxy-addr", cfg.ProxyServer.Address),
		slog.String("api-addr", cfg.ApiServer.Address),
	)
	log.Debug("debug messages are enabled")

	db, err := gorm.Open(sqlite.Open("./stage.db"), &gorm.Config{})
	if err != nil {
		log.Error("Error connect to storage", sl.Err(err))
		os.Exit(1)
	}
	db.AutoMigrate(&models.RequestResponse{})
	repoRequest := storage.NewRequestsRepo(db)

	cm, err := proxy.NewCertManager(cfg.Cert)
	if err != nil {
		log.Error("Failed init cert manager", sl.Err(err))
		os.Exit(1)
	}

	srvProxy := &http.Server{
		Handler: proxy.New(
			log,
			cm,
			repoRequest),
		Addr:              cfg.ProxyServer.Address,
		ReadTimeout:       cfg.ProxyServer.ReadTimeout,
		WriteTimeout:      cfg.ProxyServer.WriteTimeout,
		IdleTimeout:       cfg.ProxyServer.IdleTimeout,
		ReadHeaderTimeout: cfg.ProxyServer.ReadHeaderTimeout,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler)), //disable http2
	}

	srvApi := &http.Server{
		Addr:              cfg.ApiServer.Address,
		ReadTimeout:       cfg.ApiServer.ReadTimeout,
		WriteTimeout:      cfg.ApiServer.WriteTimeout,
		IdleTimeout:       cfg.ApiServer.IdleTimeout,
		ReadHeaderTimeout: cfg.ApiServer.ReadHeaderTimeout,
	}

	//= = = = = = = Echo for API = = = = = = =//
	e := echo.New()

	e.Debug = true
	e.HideBanner = true
	e.Server = srvApi
	///requests – список запросов
	//requests/id – вывод 1 запроса
	//repeat/id – повторная отправка запроса
	//scan/id – сканирование запроса)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "You <----> TrueProxy <----> Wild Network")
	})

	e.GET("/requests", list.New(log, repoRequest))

	//- - - - - - - Echo for API - - - - - - -//

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// EXPOSE PORT 62801
		if err := srvProxy.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Error("proxy server returned err: ", sl.Err(err))
		}
	}()

	go func() {
		// EXPOSE PORT 62802
		err := e.Start(cfg.ApiServer.Address)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed while running http api server", sl.Err(err))
		}

	}()

	log.Debug("server started")
	sig := <-quit
	log.Debug("handle quit chanel: ", slog.Any("os.Signal", sig.String()))
	log.Debug("server stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShotdownTimeout)
	defer cancel()

	if err := srvProxy.Shutdown(ctx); err != nil {
		log.Error("server shutdown returned an err: ", sl.Err(err))
	}
	if err := srvApi.Shutdown(ctx); err != nil {
		log.Error("server shutdown returned an err: ", sl.Err(err))
	}

	log.Debug("server stopped")

}
