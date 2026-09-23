package api

import (
	"strconv"
	"strings"
	"time"

	"caoyou/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *Server) handleAdminStats(c *gin.Context) {
	type stat struct {
		Users     int64 `json:"users"`
		Mailboxes int64 `json:"mailboxes"`
		Messages  int64 `json:"messages"`
		Sent      int64 `json:"sent"`
		Today     int64 `json:"today"`
	}
	var st stat
	s.db.Model(&store.User{}).Count(&st.Users)
	s.db.Model(&store.Mailbox{}).Count(&st.Mailboxes)
	s.db.Model(&store.Message{}).Where("folder = ?", "inbox").Count(&st.Messages)
	s.db.Model(&store.Message{}).Where("folder = ?", "sent").Count(&st.Sent)
	s.db.Model(&store.Message{}).Where("created_at >= ?", time.Now().Truncate(24*time.Hour)).Count(&st.Today)
	c.JSON(200, gin.H{"data": st})
}

func (s *Server) handleAdminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	q := strings.TrimSpace(c.Query("q"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	dbQ := s.db.Model(&store.User{})
	if q != "" {
		dbQ = dbQ.Where("username LIKE ?", "%"+q+"%")
	}
	var total int64
	dbQ.Count(&total)
	var users []store.User
	dbQ.Order("id").Offset((page - 1) * size).Limit(size).Find(&users)
	// 附带每个用户的邮箱数
	type userRow struct {
		store.User
		MailboxCount int64 `json:"mailbox_count"`
	}
	rows := make([]userRow, 0, len(users))
	for _, u := range users {
		var cnt int64
		s.db.Model(&store.Mailbox{}).Where("user_id = ?", u.ID).Count(&cnt)
		rows = append(rows, userRow{u, cnt})
	}
	c.JSON(200, gin.H{"data": gin.H{"total": total, "items": rows}})
}

func (s *Server) adminUserByID(c *gin.Context) (*store.User, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return nil, false
	}
	var u store.User
	if err := s.db.First(&u, id).Error; err != nil {
		fail(c, 404, "用户不存在")
		return nil, false
	}
	return &u, true
}

func (s *Server) handleAdminDisableUser(c *gin.Context) {
	u, ok := s.adminUserByID(c)
	if !ok {
		return
	}
	if u.ID == currentUser(c).ID {
		fail(c, 400, "不能禁用自己")
		return
	}
	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	s.db.Model(u).Update("disabled", req.Disabled)
	c.JSON(200, gin.H{"data": "ok"})
}

func (s *Server) handleAdminResetPassword(c *gin.Context) {
	u, ok := s.adminUserByID(c)
	if !ok {
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		failBind(c, err)
		return
	}
	if len(req.Password) < 6 {
		fail(c, 400, "密码至少6位")
		return
	}
	hash, err := store.HashPassword(req.Password)
	if err != nil {
		fail(c, 500, "服务器错误")
		return
	}
	s.db.Model(u).Update("password_hash", hash)
	c.JSON(200, gin.H{"data": "ok"})
}

func (s *Server) handleAdminDeleteUser(c *gin.Context) {
	u, ok := s.adminUserByID(c)
	if !ok {
		return
	}
	if u.ID == currentUser(c).ID {
		fail(c, 400, "不能删除自己")
		return
	}
	var mbIDs []uint
	s.db.Model(&store.Mailbox{}).Where("user_id = ?", u.ID).Pluck("id", &mbIDs)
	if len(mbIDs) > 0 {
		var msgIDs []uint
		s.db.Model(&store.Message{}).Where("mailbox_id IN ?", mbIDs).Pluck("id", &msgIDs)
		if len(msgIDs) > 0 {
			s.db.Where("message_id IN ?", msgIDs).Delete(&store.Attachment{})
		}
		s.db.Where("mailbox_id IN ?", mbIDs).Delete(&store.Message{})
		s.db.Where("id IN ?", mbIDs).Delete(&store.Mailbox{})
	}
	s.db.Delete(u)
	c.JSON(200, gin.H{"data": "ok"})
}

func (s *Server) handleAdminMailboxes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var total int64
	s.db.Model(&store.Mailbox{}).Count(&total)
	var mbs []store.Mailbox
	s.db.Order("id").Offset((page - 1) * size).Limit(size).Find(&mbs)
	// 附带属主用户名
	type mbRow struct {
		store.Mailbox
		Owner string `json:"owner"`
	}
	rows := make([]mbRow, 0, len(mbs))
	for _, mb := range mbs {
		rows = append(rows, mbRow{mb, ownerName(s.db, mb.UserID)})
	}
	c.JSON(200, gin.H{"data": gin.H{"total": total, "items": rows}})
}

func ownerName(db *gorm.DB, userID uint) string {
	var u store.User
	if err := db.First(&u, userID).Error; err != nil {
		return "(已删除)"
	}
	return u.Username
}

func (s *Server) handleAdminDeleteMailbox(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		fail(c, 400, "无效ID")
		return
	}
	var mb store.Mailbox
	if err := s.db.First(&mb, id).Error; err != nil {
		fail(c, 404, "邮箱不存在")
		return
	}
	var msgIDs []uint
	s.db.Model(&store.Message{}).Where("mailbox_id = ?", mb.ID).Pluck("id", &msgIDs)
	if len(msgIDs) > 0 {
		s.db.Where("message_id IN ?", msgIDs).Delete(&store.Attachment{})
	}
	s.db.Where("mailbox_id = ?", mb.ID).Delete(&store.Message{})
	s.db.Delete(&mb)
	c.JSON(200, gin.H{"data": "ok"})
}
