package api

import (
	"strings"

	"caoyou/internal/mailer"
	"caoyou/internal/store"

	"github.com/gin-gonic/gin"
)

// viewOf 转前端视图，不回传密码/私钥明文
func viewOf(ms *mailer.MailSettings) gin.H {
	return gin.H{
		"domain":         ms.Domain,
		"mode":           ms.Mode,
		"helo_domain":    ms.HeloDomain,
		"relay_host":     ms.RelayHost,
		"relay_port":     ms.RelayPort,
		"relay_tls":      ms.RelayTLS,
		"relay_user":     ms.RelayUser,
		"has_relay_pass": ms.RelayPassEnc != "",
	}
}

// dkimDNSOf 计算DKIM的DNS记录信息
func dkimDNSOf(ms *mailer.MailSettings) gin.H {
	return gin.H{
		"name": ms.DKIMSelector + "._domainkey." + ms.Domain,
		"txt":  dkimTXTOf(ms),
	}
}

func dkimTXTOf(ms *mailer.MailSettings) string {
	return ms.DKIMDNSTXT()
}

func (s *Server) handleGetMailSettings(c *gin.Context) {
	ms, err := mailer.LoadMailSettings(s.db, s.cfg)
	if err != nil {
		fail(c, 500, "读取设置失败")
		return
	}
	v := viewOf(ms)
	v["dkim_dns"] = dkimDNSOf(ms)
	c.JSON(200, gin.H{"data": v})
}

func (s *Server) handleSaveMailSettings(c *gin.Context) {
	ms, err := mailer.LoadMailSettings(s.db, s.cfg)
	if err != nil {
		fail(c, 500, "读取设置失败")
		return
	}
	var req struct {
		Domain       string `json:"domain"`
		Mode         string `json:"mode"`
		HeloDomain   string `json:"helo_domain"`
		RelayHost    string `json:"relay_host"`
		RelayPort    int    `json:"relay_port"`
		RelayTLS     string `json:"relay_tls"`
		RelayUser    string `json:"relay_user"`
		RelayPass    string `json:"relay_pass"`
		DKIMEnabled  bool   `json:"dkim_enabled"`
		DKIMSelector string `json:"dkim_selector"`
		DKIMKeyPEM   string `json:"dkim_key_pem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" || strings.Contains(domain, " ") || !strings.Contains(domain, ".") && domain != "localhost" {
		fail(c, 400, "域名格式不正确")
		return
	}
	// 域名变更时，本地邮箱地址同步改为新域名后缀（外部邮箱不动；新地址已存在的跳过）
	if oldDomain := strings.ToLower(strings.TrimSpace(ms.Domain)); oldDomain != domain {
		var mbs []store.Mailbox
		s.db.Where("type = ?", "local").Find(&mbs)
		for _, mb := range mbs {
			if !strings.HasSuffix(mb.Address, "@"+oldDomain) {
				continue
			}
			local := strings.SplitN(mb.Address, "@", 2)[0]
			newAddr := local + "@" + domain
			var cnt int64
			s.db.Model(&store.Mailbox{}).Where("address = ?", newAddr).Count(&cnt)
			if cnt == 0 {
				s.db.Model(&store.Mailbox{}).Where("id = ?", mb.ID).Update("address", newAddr)
			}
		}
	}
	ms.Domain = domain
	ms.Mode = req.Mode
	if ms.Mode != "relay" && ms.Mode != "direct" {
		ms.Mode = "relay"
	}
	ms.HeloDomain = strings.TrimSpace(req.HeloDomain)
	if ms.HeloDomain == "" {
		ms.HeloDomain = ms.Domain
	}
	ms.RelayHost = strings.TrimSpace(req.RelayHost)
	ms.RelayPort = req.RelayPort
	if ms.RelayPort <= 0 {
		ms.RelayPort = 465
	}
	ms.RelayTLS = req.RelayTLS
	if ms.RelayTLS != "ssl" && ms.RelayTLS != "starttls" && ms.RelayTLS != "none" {
		ms.RelayTLS = "ssl"
	}
	ms.RelayUser = strings.TrimSpace(req.RelayUser)
	if req.RelayPass != "" {
		ms.RelayPassEnc = store.Encrypt(req.RelayPass, s.cfg.AESKey)
	}
	ms.DKIMEnabled = req.DKIMEnabled
	ms.DKIMSelector = strings.ToLower(strings.TrimSpace(req.DKIMSelector))
	if req.DKIMKeyPEM != "" && strings.Contains(req.DKIMKeyPEM, "PRIVATE KEY") {
		ms.DKIMKeyPEM = req.DKIMKeyPEM
	}
	if err := mailer.SaveMailSettings(s.db, ms); err != nil {
		fail(c, 500, "保存失败")
		return
	}
	c.JSON(200, gin.H{"data": "ok"})
}

// handleGenerateDKIM 生成DKIM密钥对并保存私钥，返回DNS配置说明
func (s *Server) handleGenerateDKIM(c *gin.Context) {
	var req struct {
		Selector string `json:"selector"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	selector := strings.ToLower(strings.TrimSpace(req.Selector))
	if selector == "" {
		selector = "caoyou"
	}
	ms, err := mailer.LoadMailSettings(s.db, s.cfg)
	if err != nil {
		fail(c, 500, "读取设置失败")
		return
	}
	pemKey, dnsTXT, err := mailer.GenerateDKIMKey()
	if err != nil {
		fail(c, 500, "密钥生成失败")
		return
	}
	ms.DKIMSelector = selector
	ms.DKIMKeyPEM = pemKey
	mailer.SaveMailSettings(s.db, ms)
	c.JSON(200, gin.H{"data": gin.H{
		"selector": selector,
		"key_pem":  pemKey,
		"dns_name": selector + "._domainkey." + ms.Domain,
		"dns_txt":  dnsTXT,
		"note":     "请到DNS服务商添加TXT记录：名称=selector._domainkey.你的域名，内容=上方txt值",
	}})
}
