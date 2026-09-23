package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"caoyou/internal/api"
	"caoyou/internal/config"
	"caoyou/internal/fetcher"
	"caoyou/internal/mailer"
	"caoyou/internal/smtpsrv"
	"caoyou/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	cfgPath := flag.String("c", "configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("加载配置失败", "err", err)
		os.Exit(1)
	}
	db, err := store.Open(cfg.DB.Driver, cfg.DB.DSN)
	if err != nil {
		slog.Error("连接数据库失败", "err", err)
		os.Exit(1)
	}
	if err := store.AutoMigrate(db); err != nil {
		slog.Error("数据库迁移失败", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.SMTPIn.Enabled {
		srv := smtpsrv.New(db, func() string {
			ms, _ := mailer.LoadMailSettings(db, cfg)
			return ms.Domain
		}, int64(cfg.SMTPIn.MaxMB))
		go func() {
			slog.Info("SMTP收信服务已启动", "addr", cfg.SMTPIn.Addr)
			if err := srv.ListenAndServe(cfg.SMTPIn.Addr); err != nil {
				slog.Error("SMTP服务退出", "err", err)
			}
		}()
	}
	go fetcher.Run(ctx, db, time.Duration(cfg.FetchIntervalSec)*time.Second, cfg.AESKey)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.MaxMultipartMemory = 64 << 20
	api.Register(r, db, cfg)
	api.ServeSPA(r, cfg.WebDist)

	go func() {
		if err := r.Run(cfg.Server.Addr); err != nil {
			slog.Error("HTTP服务退出", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("服务已停止")
}
