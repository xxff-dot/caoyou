package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"caoyou/internal/mailer"
	"caoyou/internal/store"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"gorm.io/gorm"
)

// Run 周期性拉取所有外部邮箱
func Run(ctx context.Context, db *gorm.DB, interval time.Duration, aesKey string) {
	runAll(db, aesKey)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runAll(db, aesKey)
		}
	}
}

func runAll(db *gorm.DB, aesKey string) {
	var mbs []store.Mailbox
	if err := db.Where("type = ?", "external").Find(&mbs).Error; err != nil {
		slog.Error("fetcher: query mailboxes", "err", err)
		return
	}
	for _, mb := range mbs {
		if err := fetchOne(db, &mb, aesKey); err != nil {
			slog.Warn("fetcher", "addr", mb.Address, "err", err)
			db.Model(&store.Mailbox{}).Where("id = ?", mb.ID).
				Update("last_error", err.Error())
		} else {
			db.Model(&store.Mailbox{}).Where("id = ?", mb.ID).
				Update("last_error", "")
		}
	}
}

func fetchOne(db *gorm.DB, mb *store.Mailbox, aesKey string) error {
	pass, err := store.Decrypt(mb.Password, aesKey)
	if err != nil {
		return err
	}
	c, err := imapclient.DialTLS(fmt.Sprintf("%s:%d", mb.ImapHost, mb.ImapPort), nil)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Login(mb.Address, pass).Wait(); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	sel, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		return fmt.Errorf("select INBOX: %w", err)
	}
	n := sel.NumMessages
	if n == 0 {
		return nil
	}
	start := uint32(1)
	if n > 50 {
		start = n - 49
	}
	seq := imap.SeqSet{}
	seq.AddRange(start, n)
	fetched, err := c.Fetch(seq, &imap.FetchOptions{
		UID:         true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}).Collect()
	if err != nil {
		return err
	}
	var maxUID uint32
	for _, msg := range fetched {
		u := uint32(msg.UID)
		if u > maxUID {
			maxUID = u
		}
		if u <= mb.LastUID {
			continue
		}
		raw := msg.FindBodySection(&imap.FetchItemBodySection{Peek: true})
		if len(raw) == 0 {
			continue
		}
		if err := mailer.ParseAndStore(db, mb.ID, "inbox", "", "", raw); err != nil {
			slog.Warn("fetcher: store message", "addr", mb.Address, "err", err)
		}
	}
	if maxUID > mb.LastUID {
		return db.Model(mb).Update("last_uid", maxUID).Error
	}
	return nil
}
