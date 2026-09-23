package api

import (
	"io"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"caoyou/internal/config"
	"caoyou/internal/mailer"
	"caoyou/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const cookieName = "caoyou_token"

var (
	usernameRe  = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	localPartRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	emailRe     = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type Server struct {
	db  *gorm.DB
	cfg *config.Config
}

func Register(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	s := &Server{db: db, cfg: cfg}
	a := r.Group("/api")
	a.POST("/register", s.handleRegister)
	a.POST("/login", s.handleLogin)

	auth := a.Group("", s.authRequired())
	auth.POST("/logout", s.handleLogout)
	auth.GET("/me", s.handleMe)
	auth.GET("/config", s.handleConfig)
	auth.GET("/stats", s.handleStats)

	auth.GET("/mailboxes", s.handleListMailboxes)
	auth.POST("/mailboxes", s.handleCreateMailbox)
	auth.POST("/external", s.handleBindExternal)
	auth.DELETE("/mailboxes/:id", s.handleDeleteMailbox)
	auth.GET("/mailboxes/:id/messages", s.handleListMessages)
	auth.GET("/mailboxes/:id/unread", s.handleUnreadCount)

	auth.GET("/messages/:id", s.handleGetMessage)
	auth.DELETE("/messages/:id", s.handleDeleteMessage)
	auth.GET("/attachments/:id", s.handleDownloadAttachment)
	auth.POST("/send", s.handleSend)

	adminGroup := a.Group("/admin", s.authRequired(), adminRequired())
	adminGroup.GET("/stats", s.handleAdminStats)
	adminGroup.GET("/users", s.handleAdminUsers)
	adminGroup.POST("/users/:id/disable", s.handleAdminDisableUser)
	adminGroup.POST("/users/:id/resetpw", s.handleAdminResetPassword)
	adminGroup.DELETE("/users/:id", s.handleAdminDeleteUser)
	adminGroup.GET("/mailboxes", s.handleAdminMailboxes)
	adminGroup.DELETE("/mailboxes/:id", s.handleAdminDeleteMailbox)
	adminGroup.GET("/settings/mail", s.handleGetMailSettings)
	adminGroup.PUT("/settings/mail", s.handleSaveMailSettings)
	adminGroup.POST("/dkim/generate", s.handleGenerateDKIM)
}

func (s *Server) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, err := c.Cookie(cookieName)
		if err != nil || tok == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		uid, err := store.ParseToken(tok, s.cfg.Server.Secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录已过期，请重新登录"})
			return
		}
		var u store.User
		if err := s.db.First(&u, uid).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			return
		}
		if u.Disabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
			return
		}
		c.Set("user", &u)
		c.Next()
	}
}

func currentUser(c *gin.Context) *store.User {
	u, _ := c.MustGet("user").(*store.User)
	return u
}

func adminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !currentUser(c).IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

func failBind(c *gin.Context, err error) {
	fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
}

func (s *Server) handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	if !usernameRe.MatchString(req.Username) {
		fail(c, 400, "用户名3-32位，仅字母数字下划线")
		return
	}
	if len(req.Password) < 6 {
		fail(c, 400, "密码至少6位")
		return
	}
	var cnt int64
	s.db.Model(&store.User{}).Where("username = ?", req.Username).Count(&cnt)
	if cnt > 0 {
		fail(c, 400, "用户名已存在")
		return
	}
	hash, err := store.HashPassword(req.Password)
	if err != nil {
		fail(c, 500, "服务器错误")
		return
	}
	var userCnt int64
	s.db.Model(&store.User{}).Count(&userCnt)
	u := store.User{Username: req.Username, PasswordHash: hash, IsAdmin: userCnt == 0}
	if err := s.db.Create(&u).Error; err != nil {
		fail(c, 500, "服务器错误")
		return
	}
	// 注册即自动开通 用户名@域名（被占用则跳过，可稍后手动创建）
	ms, _ := mailer.LoadMailSettings(s.db, s.cfg)
	if local := strings.ToLower(req.Username); localPartRe.MatchString(local) {
		addr := local + "@" + ms.Domain
		var mbCnt int64
		s.db.Model(&store.Mailbox{}).Where("address = ?", addr).Count(&mbCnt)
		if mbCnt == 0 {
			s.db.Create(&store.Mailbox{UserID: u.ID, Type: "local", Address: addr})
		}
	}
	s.setSession(c, u.ID)
	c.JSON(200, gin.H{"data": u})
}

func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	var u store.User
	if err := s.db.Where("username = ?", req.Username).First(&u).Error; err != nil {
		fail(c, 400, "用户名或密码错误")
		return
	}
	if !store.CheckPassword(u.PasswordHash, req.Password) {
		fail(c, 400, "用户名或密码错误")
		return
	}
	s.setSession(c, u.ID)
	c.JSON(200, gin.H{"data": u})
}

func (s *Server) setSession(c *gin.Context, userID uint) {
	tok, _ := store.NewToken(userID, s.cfg.Server.Secret, 7*24*time.Hour)
	// ponytail: secure=false 便于本地HTTP调试；上HTTPS时改为true
	c.SetCookie(cookieName, tok, 7*24*3600, "/", "", false, true)
}

func (s *Server) handleLogout(c *gin.Context) {
	c.SetCookie(cookieName, "", -1, "/", "", false, true)
	c.JSON(200, gin.H{"data": "ok"})
}

func (s *Server) handleMe(c *gin.Context) {
	c.JSON(200, gin.H{"data": currentUser(c)})
}

func (s *Server) handleConfig(c *gin.Context) {
	ms, _ := mailer.LoadMailSettings(s.db, s.cfg)
	c.JSON(200, gin.H{"data": gin.H{"domain": ms.Domain}})
}

func (s *Server) handleStats(c *gin.Context) {
	u := currentUser(c)
	type stat struct {
		Mailboxes int64 `json:"mailboxes"`
		Messages  int64 `json:"messages"`
		Unread    int64 `json:"unread"`
		Sent      int64 `json:"sent"`
		Today     int64 `json:"today"`
	}
	var st stat
	s.db.Model(&store.Mailbox{}).Where("user_id = ?", u.ID).Count(&st.Mailboxes)
	msgQ := s.db.Model(&store.Message{}).Joins("JOIN mailboxes ON mailboxes.id = messages.mailbox_id").Where("mailboxes.user_id = ?", u.ID)
	msgQ.Session(&gorm.Session{}).Where("folder = ?", "inbox").Count(&st.Messages)
	msgQ.Session(&gorm.Session{}).Where("folder = ? AND is_read = ?", "inbox", false).Count(&st.Unread)
	msgQ.Session(&gorm.Session{}).Where("folder = ?", "sent").Count(&st.Sent)
	msgQ.Session(&gorm.Session{}).Where("messages.created_at >= ?", time.Now().Truncate(24*time.Hour)).Count(&st.Today)
	c.JSON(200, gin.H{"data": st})
}

func (s *Server) ownMailbox(c *gin.Context) (*store.Mailbox, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return nil, false
	}
	var mb store.Mailbox
	if err := s.db.Where("id = ? AND user_id = ?", id, currentUser(c).ID).First(&mb).Error; err != nil {
		fail(c, 404, "邮箱不存在")
		return nil, false
	}
	return &mb, true
}

// accessMailbox 属主或管理员可访问（用于读信）
func (s *Server) accessMailbox(c *gin.Context) (*store.Mailbox, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return nil, false
	}
	var mb store.Mailbox
	q := s.db.Where("id = ?", id)
	if !currentUser(c).IsAdmin {
		q = q.Where("user_id = ?", currentUser(c).ID)
	}
	if err := q.First(&mb).Error; err != nil {
		fail(c, 404, "邮箱不存在")
		return nil, false
	}
	return &mb, true
}

// canReadMailbox 消息级权限：属主或管理员
func canReadMailbox(mb *store.Mailbox, u *store.User) bool {
	return mb.UserID == u.ID || u.IsAdmin
}

func (s *Server) handleListMailboxes(c *gin.Context) {
	var mbs []store.Mailbox
	s.db.Where("user_id = ?", currentUser(c).ID).Order("id").Find(&mbs)
	c.JSON(200, gin.H{"data": mbs})
}

func (s *Server) handleCreateMailbox(c *gin.Context) {
	var req struct {
		Local string `json:"local"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	local := strings.ToLower(strings.TrimSpace(req.Local))
	if !localPartRe.MatchString(local) {
		fail(c, 400, "地址仅限小写字母、数字和._-，最长64位")
		return
	}
	ms, _ := mailer.LoadMailSettings(s.db, s.cfg)
	addr := local + "@" + ms.Domain
	var cnt int64
	s.db.Model(&store.Mailbox{}).Where("address = ?", addr).Count(&cnt)
	if cnt > 0 {
		fail(c, 400, "该地址已被占用")
		return
	}
	mb := store.Mailbox{UserID: currentUser(c).ID, Type: "local", Address: addr}
	if err := s.db.Create(&mb).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}
	c.JSON(200, gin.H{"data": mb})
}

func (s *Server) handleBindExternal(c *gin.Context) {
	var req struct {
		Address  string `json:"address"`
		Password string `json:"password"`
		ImapHost string `json:"imap_host"`
		ImapPort int    `json:"imap_port"`
		SmtpHost string `json:"smtp_host"`
		SmtpPort int    `json:"smtp_port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	addr := strings.ToLower(strings.TrimSpace(req.Address))
	if !emailRe.MatchString(addr) {
		fail(c, 400, "邮箱地址格式不正确")
		return
	}
	if req.ImapHost == "" || req.SmtpHost == "" {
		fail(c, 400, "IMAP/SMTP服务器不能为空")
		return
	}
	if req.ImapPort <= 0 {
		req.ImapPort = 993
	}
	if req.SmtpPort <= 0 {
		req.SmtpPort = 465
	}
	mb := store.Mailbox{
		UserID:   currentUser(c).ID,
		Type:     "external",
		Address:  addr,
		Password: store.Encrypt(req.Password, s.cfg.AESKey),
		ImapHost: req.ImapHost,
		ImapPort: req.ImapPort,
		SmtpHost: req.SmtpHost,
		SmtpPort: req.SmtpPort,
	}
	if err := s.db.Create(&mb).Error; err != nil {
		fail(c, 400, "该邮箱已绑定或保存失败")
		return
	}
	c.JSON(200, gin.H{"data": mb})
}

func (s *Server) handleDeleteMailbox(c *gin.Context) {
	mb, ok := s.ownMailbox(c)
	if !ok {
		return
	}
	var msgIDs []uint
	s.db.Model(&store.Message{}).Where("mailbox_id = ?", mb.ID).Pluck("id", &msgIDs)
	if len(msgIDs) > 0 {
		s.db.Where("message_id IN ?", msgIDs).Delete(&store.Attachment{})
	}
	s.db.Where("mailbox_id = ?", mb.ID).Delete(&store.Message{})
	s.db.Delete(mb)
	c.JSON(200, gin.H{"data": "ok"})
}

func (s *Server) handleListMessages(c *gin.Context) {
	mb, ok := s.accessMailbox(c)
	if !ok {
		return
	}
	folder := c.DefaultQuery("folder", "inbox")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if size < 1 || size > 100 {
		size = 20
	}
	if page < 1 {
		page = 1
	}
	var total int64
	q := s.db.Model(&store.Message{}).Where("mailbox_id = ? AND folder = ?", mb.ID, folder)
	q.Count(&total)
	var items []store.Message
	s.db.Where("mailbox_id = ? AND folder = ?", mb.ID, folder).
		Select("id, mailbox_id, folder, from_addr, to_addr, subject, is_read, sent_at, substr(text_body, 1, 120) AS snippet, (SELECT COUNT(*) FROM attachments att WHERE att.message_id = messages.id) AS attachment_count").
		Order("sent_at DESC, id DESC").
		Offset((page - 1) * size).Limit(size).Find(&items)
	for i := range items {
		if items[i].Snippet == "" {
			items[i].Snippet = "(HTML邮件，点击查看)"
		}
	}
	c.JSON(200, gin.H{"data": gin.H{"total": total, "items": items}})
}

func (s *Server) handleUnreadCount(c *gin.Context) {
	mb, ok := s.ownMailbox(c)
	if !ok {
		return
	}
	var cnt int64
	s.db.Model(&store.Message{}).Where("mailbox_id = ? AND folder = ? AND is_read = ?", mb.ID, "inbox", false).Count(&cnt)
	c.JSON(200, gin.H{"data": cnt})
}

func (s *Server) handleGetMessage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return
	}
	var msg store.Message
	if err := s.db.First(&msg, id).Error; err != nil {
		fail(c, 404, "邮件不存在")
		return
	}
	var mb store.Mailbox
	if err := s.db.First(&mb, msg.MailboxID).Error; err != nil || !canReadMailbox(&mb, currentUser(c)) {
		fail(c, 404, "邮件不存在")
		return
	}
	if !msg.IsRead {
		s.db.Model(&msg).Update("is_read", true)
	}
	var atts []store.Attachment
	s.db.Where("message_id = ?", msg.ID).Find(&atts)
	c.JSON(200, gin.H{"data": gin.H{"message": msg, "attachments": atts}})
}

func (s *Server) handleDeleteMessage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return
	}
	var msg store.Message
	if err := s.db.First(&msg, id).Error; err != nil {
		fail(c, 404, "邮件不存在")
		return
	}
	var mb store.Mailbox
	if err := s.db.First(&mb, msg.MailboxID).Error; err != nil || !canReadMailbox(&mb, currentUser(c)) {
		fail(c, 404, "邮件不存在")
		return
	}
	s.db.Where("message_id = ?", msg.ID).Delete(&store.Attachment{})
	s.db.Delete(&msg)
	c.JSON(200, gin.H{"data": "ok"})
}

func (s *Server) handleDownloadAttachment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return
	}
	var att store.Attachment
	if err := s.db.First(&att, id).Error; err != nil {
		fail(c, 404, "附件不存在")
		return
	}
	var msg store.Message
	if err := s.db.First(&msg, att.MessageID).Error; err != nil {
		fail(c, 404, "邮件不存在")
		return
	}
	var mb store.Mailbox
	if err := s.db.First(&mb, msg.MailboxID).Error; err != nil || !canReadMailbox(&mb, currentUser(c)) {
		fail(c, 404, "附件不存在")
		return
	}
	disp := mime.FormatMediaType("attachment", map[string]string{"filename": att.Filename})
	c.Header("Content-Disposition", disp)
	c.Data(http.StatusOK, "application/octet-stream", att.Content)
}

func (s *Server) handleSend(c *gin.Context) {
	mbID, err := strconv.Atoi(c.PostForm("mailbox_id"))
	if err != nil {
		fail(c, 400, "请选择发件地址")
		return
	}
	var mb store.Mailbox
	if err := s.db.Where("id = ? AND user_id = ?", mbID, currentUser(c).ID).First(&mb).Error; err != nil {
		fail(c, 404, "发件地址不存在")
		return
	}
	to := c.PostForm("to")
	subject := c.PostForm("subject")
	text := c.PostForm("text")
	html := c.PostForm("html")
	if strings.TrimSpace(to) == "" {
		fail(c, 400, "收件人不能为空")
		return
	}
	var tos []string
	for _, t := range strings.FieldsFunc(to, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' }) {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			tos = append(tos, t)
		}
	}
	if len(tos) == 0 {
		fail(c, 400, "收件人地址无效")
		return
	}
	var files []mailer.File
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for _, fh := range form.File["files"] {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				continue
			}
			files = append(files, mailer.File{Filename: fh.Filename, Content: data})
		}
	}
	errs := mailer.Send(s.db, s.cfg, &mb, mb.Address, tos, subject, text, html, files)
	c.JSON(200, gin.H{"data": gin.H{"errors": errs}})
}

func ServeSPA(r *gin.Engine, distDir string) {
	if _, err := os.Stat(distDir); err != nil {
		return
	}
	r.Static("/assets", distDir+"/assets")
	r.StaticFile("/favicon.ico", distDir+"/favicon.ico")
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.File(distDir + "/index.html")
	})
}
