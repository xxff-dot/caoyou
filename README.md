# 草鼬 CaoYou Mail

> 仓库：https://github.com/xxff-dot/caoyou

Go + Vue3 自建邮件服务：**收信（自建SMTP）、发信（relay / MX直投）、用户注册即开通邮箱、管理员后台**，网页全功能管理。

```
用户 ──注册──▶ 草鼬 ──自动开通──▶ zhangsan@example.com
外部邮件 ──互联网──▶ MX:mail.example.com:25 ──▶ 草鼬SMTP收信 ──▶ 数据库 ──▶ 网页收件箱
草鼬发信 ──▶ relay模式(经QQ等SMTP) 或 direct模式(解析对方MX直投，可加DKIM签名)
```

## 功能

- **用户系统**：注册 / 登录（bcrypt + HMAC签名Cookie），**注册即自动开通 `用户名@example.com` 邮箱**
- **邮箱地址**：本地域名真实邮箱，可创建多个；域名变更时已有本地地址自动跟随改名
- **收信**：自建SMTP服务（go-smtp），仅接收本域收件人、拒绝开放中继；HTML沙箱渲染、附件下载
- **发信**：
  - `relay` 模式：经外部SMTP账号发信（QQ/163/Gmail等，推荐）
  - `direct` 模式：解析收件域MX直投，支持 **DKIM 签名**（网页一键生成密钥）
- **外部邮箱**：绑定QQ/Gmail等，IMAP定时拉取进统一收件箱；发信走其自身SMTP
- **管理员**：首个注册用户自动成为管理员；用户管理 / 邮箱管理 / 查看任意邮箱邮件 / 全站统计 / 发信设置（全部网页操作，保存即生效）
- **存储**：SQLite（默认）/ MySQL / PostgreSQL，GORM一套代码切换
- **前端**：Vue3 + Vite + Element Plus 企业级界面

## 目录结构

```
server/                 Go 后端
├── cmd/caoyou/         程序入口 main.go
├── configs/config.yaml 引导配置（仅首次启动的初始值）
├── internal/
│   ├── api/            REST接口（含管理员接口）
│   ├── config/         配置加载
│   ├── fetcher/        外部邮箱IMAP定时拉取
│   ├── mailer/         MIME构建/解析、发信、DKIM、网页设置模型
│   ├── smtpsrv/        自建SMTP收信服务
│   └── store/          GORM模型与存储
└── go.mod
web/                    Vue3 前端（构建产物 web/dist）
```

## 快速开始（本地开发）

```bash
# 1. 构建前端
cd web && npm install && npm run build

# 2. 启动后端（默认 SQLite，web 8080，SMTP 25，域名 localhost）
cd ../server && go build -o caoyou ./cmd/caoyou && ./caoyou
```

打开 http://localhost:8080 → 注册（自动开通 `用户名@localhost`）→ 收件箱。

本地模拟外部来信测试：

```bash
printf 'From: friend@test.com\r\nTo: zhangsan@localhost\r\nSubject: hello\r\n\r\n你好\r\n' | \
  curl smtp://localhost:25 --mail-from friend@test.com --mail-rcpt zhangsan@localhost -T -
```

## 生产部署（以 example.com 为例）

前提：一台有**公网IP**的服务器（示例IP `203.0.113.10`），域名 `example.com` 可自由解析。
> 收信必须开放**入站25端口**。国内云厂商默认封禁，需在控制台申请解封/放行；海外节点入站一般不封，出站25默认封（只收不发无需理会）。

### 1. 构建与上传

```bash
# 本地交叉编译
cd server && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o caoyou-linux ./cmd/caoyou
# 前端
cd web && npm install && npm run build
# 上传
ssh root@203.0.113.10 "mkdir -p /opt/caoyou/configs /opt/caoyou/web /opt/caoyou/data"
scp caoyou-linux root@203.0.113.10:/opt/caoyou/caoyou
scp server/configs/config.yaml root@203.0.113.10:/opt/caoyou/configs/
scp -r web/dist root@203.0.113.10:/opt/caoyou/web/
```

### 2. 服务器配置与启动

`/opt/caoyou/configs/config.yaml`（生产示例）：

```yaml
server:
  addr: ":8080"            # 网页端口，改成安全组放行的任意端口
  secret: "换成32位随机字符串"   # 登录令牌签名密钥，必须修改
domain: "example.com"
db:
  driver: "sqlite"
  dsn: "data/caoyou.db"
smtp_in:
  enabled: true
  addr: ":25"
  max_mb: 32
mail: {}                   # 发信设置全在网页配置，这里留空即可
fetch_interval_sec: 300
aes_key: "换成32位随机字符串"   # 外部邮箱密码/授权码的加密密钥，必须修改
web_dist: "web/dist"
```

systemd 服务 `/etc/systemd/system/caoyou.service`：

```ini
[Unit]
Description=CaoYou Mail
After=network.target

[Service]
WorkingDirectory=/opt/caoyou
ExecStart=/opt/caoyou/caoyou
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
```

```bash
chmod +x /opt/caoyou/caoyou
systemctl daemon-reload && systemctl enable --now caoyou
ss -tlnp | grep -E ':25 |:8080 '    # 确认两个端口都在监听
```

### 3. DNS 解析（在 example.com 的解析控制台添加）

| 类型 | 主机记录 | 记录值 | 作用 | 必需性 |
|---|---|---|---|---|
| A | `mail` | `203.0.113.10` | 邮件服务器地址 | **收信必需** |
| MX | `@` | `mail.example.com`（优先级10） | 声明 @example.com 的信送到哪 | **收信必需** |
| TXT | `@` | `v=spf1 mx -all` | SPF：允许本域MX发信 | 发信建议 |
| TXT | `caoyou._domainkey` | 网页生成的 DKIM 公钥（见下文） | 发信签名验证 | 直发建议 |
| TXT | `_dmarc` | `v=DMARC1; p=none` | DMARC 策略 | 发信建议 |

验证解析是否生效：

```bash
nslookup -type=mx example.com 223.5.5.5          # 应返回 mail.example.com
nslookup mail.example.com 223.5.5.5              # 应返回 203.0.113.10
nslookup -type=txt caoyou._domainkey.example.com 223.5.5.5
```

### 4. 防火墙/安全组

| 方向 | 端口 | 用途 |
|---|---|---|
| 入站 | 25/tcp | 接收外部邮件（**必需**，源 0.0.0.0/0） |
| 入站 | 8080/tcp（或自定义） | 网页访问 |

### 5. 收信验证

用任意外部邮箱（QQ/Gmail）发一封到 `admin@example.com` → 网页收件箱刷新即可看到，点击邮件行查看正文与附件。

收不到时按序排查：

```bash
# 服务器上先确认服务本身活着（绕过网络）
printf 'From: t@t.com\r\nTo: admin@example.com\r\nSubject: test\r\n\r\nhi\r\n' | \
  curl smtp://localhost:25 --mail-from t@t.com --mail-rcpt admin@example.com -T -
# 本地通但外部信收不到 → 安全组25未放行；查日志
journalctl -u caoyou -n 50
```

## 配置说明

配置分两层：**config.yaml 是引导配置**（数据库连接、监听地址等启动必需项），**域名与全部发信参数在网页配置**（存数据库，保存即生效，无需重启）。

### config.yaml（引导配置）

| 字段 | 说明 | 默认 |
|---|---|---|
| `server.addr` | 网页监听地址 | `:8080` |
| `server.secret` | 登录令牌签名密钥 | 无，**生产必须改** |
| `domain` | 初始邮件域名 | `localhost` |
| `db.driver` | `sqlite` / `mysql` / `postgres` | `sqlite` |
| `db.dsn` | 各数据库连接串（见下文） | `caoyou.db` |
| `smtp_in.enabled` | 是否启用自建SMTP收信 | `true` |
| `smtp_in.addr` | SMTP监听地址 | `:25` |
| `smtp_in.max_mb` | 单封邮件大小上限 | `32` |
| `mail` | 首次启动时的发信初始值，之后以网页为准 | — |
| `fetch_interval_sec` | 外部邮箱IMAP拉取间隔（秒） | `300` |
| `aes_key` | 敏感信息加密密钥 | 无，**生产必须改** |
| `web_dist` | 前端构建产物目录 | `../web/dist`（本地布局） |

### 网页设置（运行时，系统管理 → 发信设置）

**邮件域名**：改为 `example.com` 后保存即生效；所有本地邮箱地址自动改名为新域名后缀（外部绑定邮箱不受影响，新地址与已有地址冲突时跳过）。

**发信模式**：

#### relay 模式（推荐，送达率最好）

经外部SMTP账号发信。以QQ邮箱为例：

1. QQ邮箱网页版 → 设置 → 账号 → 开启「IMAP/SMTP服务」→ 获取**授权码**（不是QQ密码）
2. 网页发信设置：模式 `relay`，SMTP服务器 `smtp.qq.com`，端口 `465`，加密 `SSL`，账号填QQ邮箱，密码填授权码 → 保存
3. 其他服务商：163 → `smtp.163.com:465`；Gmail → `smtp.gmail.com:465`（需应用专用密码）

> 注意：relay 发信时发件人身份跟随所配邮箱账号，适合个人通知类邮件；要以 `xxx@example.com` 身份对外发信请用 direct 模式。

#### direct 模式（以自己域名身份直投）

解析收件域MX直投对方服务器。要求：出站25端口可用 + 以下信誉三件套：

| 项目 | 配置位置 | 说明 |
|---|---|---|
| PTR 反向解析 | VPS厂商控制台或工单 | `203.0.113.10` → `mail.example.com`，**没有PTR多数国内邮箱直接拒收** |
| SPF | DNS TXT `@` | `v=spf1 mx -all` |
| DKIM | 网页生成 + DNS TXT | 见下 |
| DMARC | DNS TXT `_dmarc` | `v=DMARC1; p=none`（观察期后可改 `quarantine`/`reject`） |

**DKIM 配置步骤**：发信设置 → 点「生成新密钥对」→ 按页面给出的记录在DNS添加
`caoyou._domainkey.example.com` 的TXT记录（内容形如 `v=DKIM1; k=rsa; p=MIIB...`）→ 打开DKIM开关 → 保存。
此后所有外发邮件自动携带 `DKIM-Signature` 头。

### 只收不发场景

若只用 `xxx@example.com` 收信：发信模式、DKIM、PTR 全部不用配。
建议DNS加两条**防伪造**记录，声明本域永不发信：

| 类型 | 主机记录 | 记录值 |
|---|---|---|
| TXT | `@` | `v=spf1 -all` |
| TXT | `_dmarc` | `v=DMARC1; p=reject` |

### 外部邮箱绑定（收Gmail/QQ等已有邮箱的信）

外部邮箱 → 绑定，或用快速预设：

| 服务商 | IMAP | SMTP | 授权码说明 |
|---|---|---|---|
| QQ邮箱 | `imap.qq.com:993` | `smtp.qq.com:465` | 设置→账号→开启IMAP/SMTP→获取授权码 |
| 163邮箱 | `imap.163.com:993` | `smtp.163.com:465` | 同上，需授权码 |
| Gmail | `imap.gmail.com:993` | `smtp.gmail.com:465` | Google账号开启两步验证后生成「应用专用密码」 |

绑定后后台每 `fetch_interval_sec` 秒拉取一次新邮件进统一收件箱；用该地址写邮件时自动走它自己的SMTP发信。邮箱密码使用 `aes_key` AES-GCM 加密存储。

## 用户指南

- **注册**：填写用户名密码 → 自动开通 `用户名@example.com`，并自动登录
- **邮箱地址**：可额外创建更多本域地址（前缀仅小写字母数字 `._-`）
- **收件箱**：顶部切换邮箱与「收件/已发送」，点击邮件行查看正文（HTML沙箱渲染）、下载附件；管理员模式下邮箱下拉框列出全站邮箱，可查看任意邮箱邮件
- **写邮件**：多收件人（逗号分隔）、多附件拖拽上传；本域收件人秒达（直接入库），外部收件人按发信模式投递，逐收件人回报失败原因
- **退出**：侧边栏底部用户卡片

## 管理员指南

系统首个注册用户自动成为管理员。侧边栏出现「系统管理」：

| 标签 | 能力 |
|---|---|
| 用户管理 | 列表（搜索/分页/邮箱数）、禁用/启用、重置密码、删除（级联删除其邮箱与邮件）；不能禁用/删除自己 |
| 邮箱管理 | 全站邮箱列表（含属主）、删除任意邮箱 |
| 系统统计 | 全站用户/邮箱/收发件量/今日新增 |
| 发信设置 | 域名、发信模式、relay账号、DKIM（见上文配置说明） |

被禁用用户所有请求立即返回「账号已被禁用」。普通用户访问管理接口返回 403。

## 数据库切换

```yaml
db:
  driver: "mysql"
  dsn: "user:pass@tcp(127.0.0.1:3306)/caoyou?charset=utf8mb4&parseTime=True&loc=Local"
  # postgres:
  # dsn: "host=127.0.0.1 port=5432 user=postgres password=xxx dbname=caoyou sslmode=disable"
```

表结构启动时自动迁移（users / mailboxes / messages / attachments / settings）。

## API 一览

所有接口前缀 `/api`，Cookie 认证（`caoyou_token`）。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /register · /login · /logout | 注册（自动开邮箱）/ 登录 / 登出 |
| GET | /me · /config · /stats | 当前用户 / 域名 / 个人统计 |
| GET · POST | /mailboxes | 列出 / 创建本域邮箱 |
| POST | /external | 绑定外部邮箱（IMAP/SMTP） |
| DELETE | /mailboxes/:id | 删除邮箱及其邮件 |
| GET | /mailboxes/:id/messages?folder=&page=&size= | 邮件列表（含预览/附件数） |
| GET | /mailboxes/:id/unread | 未读数 |
| GET · DELETE | /messages/:id | 详情（自动已读）/ 删除 |
| GET | /attachments/:id | 附件下载 |
| POST | /send (multipart) | 发送（mailbox_id, to, subject, text, files[]） |
| GET | /admin/stats | 全站统计（管理员） |
| GET | /admin/users?page&size&q | 用户列表（管理员） |
| POST | /admin/users/:id/disable | 禁用/启用（管理员） |
| POST | /admin/users/:id/resetpw | 重置密码（管理员） |
| DELETE | /admin/users/:id | 删除用户及数据（管理员） |
| GET | /admin/mailboxes | 全站邮箱（管理员） |
| DELETE | /admin/mailboxes/:id | 删除任意邮箱（管理员） |
| GET · PUT | /admin/settings/mail | 发信设置读取/保存（管理员） |
| POST | /admin/dkim/generate | 生成DKIM密钥对（管理员） |

## 常见问题

**收不到外部邮件？**
按序排查：① 服务器本地SMTP自测是否入库（见上文收信验证）→ ② 安全组/防火墙入站25是否放行 → ③ MX记录是否生效 → ④ `journalctl -u caoyou` 看有无连接日志。

**直发的信进垃圾箱/被拒收？**
依次检查：PTR反向解析（最常见的拒收原因）→ SPF → DKIM是否启用且DNS记录正确 → DMARC。新IP需逐步积累信誉。

**25端口被封？**
国内云厂商出站/入站25均默认封禁，需在控制台申请解封。收信依赖入站25（无法绕过）；发信可改 relay 模式绕开出站25。

**网页改域名后旧地址会怎样？**
所有本地邮箱地址自动改为新域名后缀；外部绑定邮箱不变；已存在的邮件头里的旧地址属历史数据不改写。

**忘记管理员密码？**
直接删库重建（SQLite场景）：停止服务 → 删除 `data/caoyou.db` → 重启 → 第一个注册的用户即新管理员。或用另一管理员账号重置。

## 测试与开发

```bash
cd server && go test ./...        # 单元测试（MIME回环、DKIM签名验证）
cd web && npm run dev             # 前端开发，/api 代理到 localhost:8080
```
