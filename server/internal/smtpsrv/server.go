package smtpsrv

import (
	"io"
	"net/mail"
	"strings"

	"caoyou/internal/mailer"
	"caoyou/internal/store"

	"github.com/emersion/go-smtp"
	"gorm.io/gorm"
)

const maxRecipients = 10

type Server struct {
	db       *gorm.DB
	domainFn func() string
	maxMB    int64
}

// New domainFn 动态返回当前域名（网页可改，即时生效）
func New(db *gorm.DB, domainFn func() string, maxMB int64) *Server {
	return &Server{
		db:       db,
		domainFn: domainFn,
		maxMB:    maxMB,
	}
}

func (s *Server) ListenAndServe(addr string) error {
	srv := smtp.NewServer(&backend{s: s})
	srv.Addr = addr
	srv.Domain = s.domainFn()
	srv.MaxMessageBytes = s.maxMB * 1024 * 1024
	srv.MaxRecipients = maxRecipients
	srv.AllowInsecureAuth = true
	return srv.ListenAndServe()
}

type backend struct {
	s *Server
}

func (b *backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &session{s: b.s}, nil
}

type session struct {
	s     *Server
	from  string
	rcpts []string
}

func (sess *session) Mail(from string, _ *smtp.MailOptions) error {
	if from == "" {
		return &smtp.SMTPError{Code: 501, Message: "sender required"}
	}
	sess.from = from
	return nil
}

func (sess *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	addr, err := mail.ParseAddress(to)
	if err != nil {
		return &smtp.SMTPError{Code: 501, Message: "bad address"}
	}
	local := strings.ToLower(addr.Address)
	i := strings.LastIndex(local, "@")
	if i < 0 || local[i+1:] != strings.ToLower(sess.s.domainFn()) {
		return &smtp.SMTPError{Code: 550, Message: "relay access denied"}
	}
	var cnt int64
	sess.s.db.Model(&store.Mailbox{}).Where("type = ? AND address = ?", "local", local).Count(&cnt)
	if cnt == 0 {
		return &smtp.SMTPError{Code: 550, Message: "mailbox not found"}
	}
	if len(sess.rcpts) >= maxRecipients {
		return &smtp.SMTPError{Code: 452, Message: "too many recipients"}
	}
	sess.rcpts = append(sess.rcpts, local)
	return nil
}

func (sess *session) Data(r io.Reader) error {
	raw, err := io.ReadAll(io.LimitReader(r, sess.s.maxMB*1024*1024+1))
	if err != nil {
		return &smtp.SMTPError{Code: 552, Message: "message too big"}
	}
	if int64(len(raw)) > sess.s.maxMB*1024*1024 {
		return &smtp.SMTPError{Code: 552, Message: "message too big"}
	}
	for _, rcpt := range sess.rcpts {
		var mb store.Mailbox
		if err := sess.s.db.Where("type = ? AND address = ?", "local", rcpt).First(&mb).Error; err != nil {
			continue
		}
		if err := mailer.ParseAndStore(sess.s.db, mb.ID, "inbox", "", "", raw); err != nil {
			return &smtp.SMTPError{Code: 451, Message: "local delivery failed"}
		}
	}
	return nil
}

func (sess *session) Reset() {
	sess.from = ""
	sess.rcpts = nil
}

func (sess *session) Logout() error {
	return nil
}
