# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 常用命令

```bash
# 后端（server/）
cd server && go build -o caoyou ./cmd/caoyou && ./caoyou   # 运行：网页 :38083，SMTP :25
cd server && go test ./...                                  # 全部测试（MIME回环、DKIM验证）
cd server && go test ./internal/mailer -run TestDKIM       # 运行单个测试

# 前端（web/）
cd web && npm install
cd web && npm run dev        # Vite :5173，/api 代理到 localhost:38083
cd web && npm run build      # 产物输出到 web/dist（由 Go 后端托管）
```

未配置 linter。`server/caoyou.db` 是开发用 SQLite 数据库，首次运行自动创建。

## 架构

单个 Go 二进制 + Vue3 SPA。一个仓库两个模块：`server/`（Go，module `caoyou`）和 `web/`（Vite）。Go 二进制直接托管 `web/dist` 的构建产物（配置项 `web_dist`），因此前端改动需 `npm run build` 后才能从后端端口看到效果。

**入口** `server/cmd/caoyou/main.go`，装配四个独立运行的部分：

1. **HTTP API**（`internal/api/`）— Gin，Cookie 认证（`caoyou_token` = HMAC 签名的 userID，密钥 `server.secret`）。`api.go` 是用户接口，`admin.go` 是管理员接口（首个注册用户自动成为管理员，`adminRequired` 中间件校验），`settings.go` 是运行时发信设置接口。`ServeSPA` 托管静态前端并处理 history 路由回退。
2. **收信 SMTP**（`internal/smtpsrv/`）— 监听 `:25`，只接收本域收件人，拒绝开放中继。注意域名通过回调从数据库设置读取，网页改域名后 SMTP 服务无需重启即生效。
3. **IMAP 拉取**（`internal/fetcher/`）— 后台协程，每 `fetch_interval_sec` 秒拉取已绑定外部邮箱的新邮件。
4. **发信**（`internal/mailer/`）— MIME 构建/解析（`build.go`，enmime）、DKIM（`dkim.go`，go-msgauth）、投递（`deliver.go`）：本域收件人直接入库；外部收件人走 `send.go`，分 `relay` 模式（外部 SMTP 账号）和 `direct` 模式（解析 MX 直投，可加 DKIM 签名）。

**存储**（`internal/store/`）— GORM 一套代码支持 sqlite/mysql/postgres（`db.Open` 按 driver 切换）。启动时 AutoMigrate；表：users、mailboxes、messages、attachments、settings。

**配置分层（重要）：** `configs/config.yaml` 仅为引导配置（数据库 DSN、监听地址、密钥）。邮件域名与全部发信设置（模式、relay 账号、DKIM 密钥）存在 `settings` 表中，在网页（系统管理 → 发信设置）修改，保存即生效 — `mailer.LoadMailSettings` 将 DB 设置合并覆盖 yaml。不要通过改 yaml 来修改运行时行为，那只会影响首次启动的初始值。`aes_key` 用于加密外部邮箱密码（AES-GCM，`store/security.go`）。

**认证模型** — bcrypt 存密码；会话为无状态 HMAC token 存 Cookie；邮箱按属主授权访问，管理员可读取/删除任意邮箱（`api.go` 的 `accessMailbox`）。

## 约定

- 后端注释、日志、API 错误信息、界面均为中文 — 保持一致。
- 新增接口：在 `api.Register` 注册路由，遵循 `fail(c, code, msg)` 错误返回模式和中间件链（`authRequired`/`adminRequired`）。
