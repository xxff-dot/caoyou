package store

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string    `gorm:"size:128" json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	Disabled     bool      `json:"disabled"`
	CreatedAt    time.Time `json:"created_at"`
}

// Mailbox 本地域名上的真实邮箱地址，或用户绑定的外部邮箱（IMAP/SMTP）
type Mailbox struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Type      string    `gorm:"size:16;index" json:"type"` // local | external
	Address   string    `gorm:"uniqueIndex;size:255" json:"address"`
	Password  string    `gorm:"size:512" json:"-"` // external: AES加密的邮箱密码
	ImapHost  string    `gorm:"size:255" json:"imap_host"`
	ImapPort  int       `json:"imap_port"`
	SmtpHost  string    `gorm:"size:255" json:"smtp_host"`
	SmtpPort  int       `json:"smtp_port"`
	LastUID   uint32    `json:"-"`
	LastError string    `gorm:"size:512" json:"last_error"`
	CreatedAt time.Time `json:"created_at"`
}

// Message 邮件。folder: inbox=收件 sent=已发送
type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MailboxID uint      `gorm:"index:idx_mb_folder" json:"mailbox_id"`
	Folder    string    `gorm:"size:16;index:idx_mb_folder" json:"folder"`
	MessageID string    `gorm:"size:998;index" json:"message_id"`
	FromName  string    `gorm:"column:from_addr;size:255" json:"from"`
	ToName    string    `gorm:"column:to_addr;size:1024" json:"to"`
	Subject   string    `json:"subject"`
	TextBody  string    `json:"text_body"`
	HTMLBody  string    `json:"html_body"`
	IsRead    bool      `json:"is_read"`
	SentAt    time.Time `json:"sent_at"`
	CreatedAt time.Time `json:"created_at"`

	// 列表查询专用（-> = 只读扫描，不落库）
	Snippet  string `gorm:"->" json:"snippet,omitempty"`
	AttCount int64  `gorm:"->" json:"attachment_count"`
}

type Attachment struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	MessageID uint   `gorm:"index" json:"message_id"`
	Filename  string `gorm:"size:255" json:"filename"`
	Size      int64  `json:"size"`
	Content   []byte `json:"-"`
}

// Setting 网页可配的系统设置（key-value，value为JSON）
type Setting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}
