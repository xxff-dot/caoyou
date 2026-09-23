package mailer

import (
	"bytes"
	"strings"
	"time"

	"caoyou/internal/store"

	"github.com/jhillyerd/enmime"
	"gorm.io/gorm"
)

// ParseAndStore 解析邮件原文并存入指定邮箱（去重：同一mailbox下相同Message-ID只存一次）
func ParseAndStore(db *gorm.DB, mailboxID uint, folder, fromOverride, toOverride string, raw []byte) error {
	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		return err
	}

	from := fromOverride
	if from == "" {
		if addrs, aerr := env.AddressList("From"); aerr == nil && len(addrs) > 0 {
			from = addrs[0].String()
		}
	}
	to := toOverride
	if to == "" {
		if addrs, aerr := env.AddressList("To"); aerr == nil && len(addrs) > 0 {
			var ss []string
			for _, a := range addrs {
				ss = append(ss, a.String())
			}
			to = strings.Join(ss, ", ")
		}
	}
	subject := env.GetHeader("Subject")
	if subject == "" {
		subject = "(无主题)"
	}
	sentAt := time.Now()
	if d, derr := env.Date(); derr == nil {
		sentAt = d
	}
	msgID := env.GetHeader("Message-Id")

	if msgID != "" {
		var cnt int64
		db.Model(&store.Message{}).Where("mailbox_id = ? AND message_id = ?", mailboxID, msgID).Count(&cnt)
		if cnt > 0 {
			return nil // 重复投递，跳过
		}
	}

	msg := store.Message{
		MailboxID: mailboxID,
		Folder:    folder,
		MessageID: msgID,
		FromName:  from,
		ToName:    to,
		Subject:   subject,
		TextBody:  env.Text,
		HTMLBody:  env.HTML,
		SentAt:    sentAt,
	}
	if err := db.Create(&msg).Error; err != nil {
		return err
	}
	for _, p := range env.Attachments {
		att := store.Attachment{
			MessageID: msg.ID,
			Filename:  p.FileName,
			Size:      int64(len(p.Content)),
			Content:   p.Content,
		}
		if err := db.Create(&att).Error; err != nil {
			return err
		}
	}
	return nil
}
