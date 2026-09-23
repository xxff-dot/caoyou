package mailer

import (
	"encoding/json"

	"caoyou/internal/config"
	"caoyou/internal/store"

	"gorm.io/gorm"
)

const settingsKey = "mail"

// MailSettings 域名+发信配置，存DB，网页可改
type MailSettings struct {
	Domain     string `json:"domain"` // 本邮件服务器域名
	Mode       string `json:"mode"`   // relay | direct
	HeloDomain string `json:"helo_domain"`
	// relay模式：经外部SMTP账号发信
	RelayHost    string `json:"relay_host"`
	RelayPort    int    `json:"relay_port"`
	RelayTLS     string `json:"relay_tls"` // ssl | starttls | none
	RelayUser    string `json:"relay_user"`
	RelayPassEnc string `json:"relay_pass_enc"` // AES加密存储
	// DKIM签名（direct模式直发时提高送达率）
	DKIMEnabled  bool   `json:"dkim_enabled"`
	DKIMSelector string `json:"dkim_selector"`
	DKIMKeyPEM   string `json:"dkim_key_pem"` // RSA私钥PEM
}

// LoadMailSettings 读取发信设置；无记录时用config.yaml初始值落库
func LoadMailSettings(db *gorm.DB, cfg *config.Config) (*MailSettings, error) {
	if raw, ok := store.GetSetting(db, settingsKey); ok {
		var ms MailSettings
		if err := json.Unmarshal([]byte(raw), &ms); err == nil {
			return &ms, nil
		}
	}
	ms := &MailSettings{
		Domain:     cfg.Domain,
		Mode:       cfg.Mail.Mode,
		HeloDomain: cfg.Mail.HeloDomain,
		RelayHost:  cfg.Mail.Relay.Host,
		RelayPort:  cfg.Mail.Relay.Port,
		RelayTLS:   cfg.Mail.Relay.TLS,
		RelayUser:  cfg.Mail.Relay.Username,
	}
	ms.RelayPassEnc = store.Encrypt(cfg.Mail.Relay.Password, cfg.AESKey)
	if err := SaveMailSettings(db, ms); err != nil {
		return nil, err
	}
	return ms, nil
}

func SaveMailSettings(db *gorm.DB, ms *MailSettings) error {
	raw, err := json.Marshal(ms)
	if err != nil {
		return err
	}
	store.SetSetting(db, settingsKey, string(raw))
	return nil
}

// RelayPass 解密后的relay密码
func (ms *MailSettings) RelayPass(aesKey string) string {
	p, _ := store.Decrypt(ms.RelayPassEnc, aesKey)
	return p
}
