package mailer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"caoyou/internal/config"
	"caoyou/internal/store"

	"gorm.io/gorm"
)

func smtpSend(host string, port int, tlsMode, username, password, from string, tos []string, raw []byte, helo string) error {
	if len(tos) == 0 {
		return errors.New("无收件人")
	}
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	var c *smtp.Client
	var err error
	switch tlsMode {
	case "ssl":
		conn, derr := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if derr != nil {
			return derr
		}
		c, err = smtp.NewClient(conn, host)
	case "starttls", "none":
		c, err = smtp.Dial(addr)
	default:
		return fmt.Errorf("未知TLS模式: %s", tlsMode)
	}
	if err != nil {
		return err
	}
	defer c.Close()

	if err = c.Hello(helo); err != nil {
		return err
	}
	if tlsMode == "starttls" {
		if err = c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if username != "" {
		if err = c.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return err
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, to := range tos {
		if err = c.Rcpt(to); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(raw); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// directSend 解析收件人域名的MX记录，直投对方邮件服务器（25端口出站需运营商放行）
func directSend(helo, from, to string, raw []byte) error {
	_, domain, found := strings.Cut(to, "@")
	if !found {
		return fmt.Errorf("非法地址: %s", to)
	}
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return fmt.Errorf("MX解析失败(%s): %w", domain, err)
	}
	var lastErr error
	for _, mx := range mxs {
		host := strings.TrimSuffix(mx.Host, ".")
		err = smtpSend(host, 25, "none", "", "", from, []string{to}, raw, helo)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func autoTLS(port int) string {
	switch port {
	case 465:
		return "ssl"
	case 587:
		return "starttls"
	default:
		return "none"
	}
}

// Send 用senderMailbox给tos发信。本地收件人直接入库，外部收件人走SMTP。
// 返回每个收件人的失败原因（成功的不在列表中）。
func Send(db *gorm.DB, cfg *config.Config, senderMailbox *store.Mailbox, from string, tos []string, subject, text, html string, files []File) map[string]error {
	ms, err := LoadMailSettings(db, cfg)
	if err != nil {
		return map[string]error{"*": err}
	}
	raw := BuildMessage("", from, tos, subject, text, html, files, ms.Domain)
	raw = SignDKIM(raw, ms)
	errs := make(map[string]error)

	for _, to := range tos {
		addr := strings.ToLower(strings.TrimSpace(to))
		if addr == "" {
			continue
		}
		if err := deliverOne(db, cfg, ms, senderMailbox, from, addr, raw); err != nil {
			errs[addr] = err
		}
	}

	// 存一份到发件箱
	ParseAndStore(db, senderMailbox.ID, "sent", from, strings.Join(tos, ", "), raw)
	return errs
}

func deliverOne(db *gorm.DB, cfg *config.Config, ms *MailSettings, senderMailbox *store.Mailbox, from, to string, raw []byte) error {
	localDomain := strings.ToLower(ms.Domain)
	if i := strings.LastIndex(to, "@"); i >= 0 && strings.EqualFold(to[i+1:], localDomain) {
		// 本地投递：直接入库
		var mb store.Mailbox
		if err := db.Where("type = ? AND address = ?", "local", to).First(&mb).Error; err != nil {
			return errors.New("本地邮箱不存在")
		}
		return ParseAndStore(db, mb.ID, "inbox", from, to, raw)
	}

	if senderMailbox.Type == "external" {
		// 用外部邮箱自己的SMTP发信
		pass, err := store.Decrypt(senderMailbox.Password, cfg.AESKey)
		if err != nil {
			return fmt.Errorf("解密邮箱密码失败: %w", err)
		}
		return smtpSend(senderMailbox.SmtpHost, senderMailbox.SmtpPort, autoTLS(senderMailbox.SmtpPort),
			senderMailbox.Address, pass, senderMailbox.Address, []string{to}, raw, cfg.Mail.HeloDomain)
	}

	switch ms.Mode {
	case "direct":
		return directSend(ms.HeloDomain, from, to, raw)
	default: // relay
		if ms.RelayHost == "" {
			return errors.New("未配置发信服务器（系统管理→发信设置）")
		}
		return smtpSend(ms.RelayHost, ms.RelayPort, ms.RelayTLS, ms.RelayUser, ms.RelayPass(cfg.AESKey), from, []string{to}, raw, ms.HeloDomain)
	}
}
