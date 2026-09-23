package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"caoyou/internal/store"
)

type File struct {
	Filename string
	Content  []byte
}

// writeTextCore 把正文（text或alternative）写入父multipart
func writeTextCore(w *multipart.Writer, text, html string) error {
	if html == "" {
		h := textproto.MIMEHeader{}
		h.Set("Content-Type", `text/plain; charset="utf-8"`)
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		pw, err := w.CreatePart(h)
		if err != nil {
			return err
		}
		qp := quotedprintable.NewWriter(pw)
		if _, err := qp.Write([]byte(text)); err != nil {
			return err
		}
		return qp.Close()
	}
	var nested bytes.Buffer
	aw := multipart.NewWriter(&nested)
	ph := textproto.MIMEHeader{}
	ph.Set("Content-Type", `text/plain; charset="utf-8"`)
	ph.Set("Content-Transfer-Encoding", "quoted-printable")
	ppw, err := aw.CreatePart(ph)
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(ppw)
	if _, err := qp.Write([]byte(text)); err != nil {
		return err
	}
	if err := qp.Close(); err != nil {
		return err
	}
	hh := textproto.MIMEHeader{}
	hh.Set("Content-Type", `text/html; charset="utf-8"`)
	hh.Set("Content-Transfer-Encoding", "quoted-printable")
	hpw, err := aw.CreatePart(hh)
	if err != nil {
		return err
	}
	qp2 := quotedprintable.NewWriter(hpw)
	if _, err := qp2.Write([]byte(html)); err != nil {
		return err
	}
	if err := qp2.Close(); err != nil {
		return err
	}
	if err := aw.Close(); err != nil {
		return err
	}
	// 作为父multipart的一个part写入
	fh := textproto.MIMEHeader{}
	fh.Set("Content-Type", `multipart/alternative; boundary="`+aw.Boundary()+`"`)
	fpw, err := w.CreatePart(fh)
	if err != nil {
		return err
	}
	_, err = fpw.Write(nested.Bytes())
	return err
}

func writeAttachment(w *multipart.Writer, f File) error {
	h := textproto.MIMEHeader{}
	ct := mime.FormatMediaType("application/octet-stream", map[string]string{"name": f.Filename})
	h.Set("Content-Type", ct)
	h.Set("Content-Transfer-Encoding", "base64")
	h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Filename}))
	pw, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	enc := base64.NewEncoder(base64.StdEncoding, &lineWrap{w: pw})
	if _, err := enc.Write(f.Content); err != nil {
		return err
	}
	enc.Close()
	return nil
}

// lineWrap base64按76列换行
type lineWrap struct {
	w io.Writer
	n int
}

func (l *lineWrap) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		chunk := 76 - l.n
		if chunk > len(p) {
			chunk = len(p)
		}
		if _, err := l.w.Write(p[:chunk]); err != nil {
			return total, err
		}
		total += chunk
		l.n += chunk
		p = p[chunk:]
		if l.n >= 76 {
			if _, err := l.w.Write([]byte("\r\n")); err != nil {
				return total, err
			}
			l.n = 0
		}
	}
	return total, nil
}

// BuildMessage 构建RFC 5322邮件
func BuildMessage(fromName, from string, tos []string, subject, text, html string, files []File, domain string) []byte {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	var topCT string
	var core func() error
	switch {
	case len(files) > 0:
		topCT = `multipart/mixed; boundary="` + w.Boundary() + `"`
		core = func() error {
			if err := writeTextCore(w, text, html); err != nil {
				return err
			}
			for _, f := range files {
				if err := writeAttachment(w, f); err != nil {
					return err
				}
			}
			return w.Close()
		}
	case html != "":
		topCT = `multipart/alternative; boundary="` + w.Boundary() + `"`
		core = func() error {
			err := writeTextCore(w, text, html)
			if err != nil {
				return err
			}
			return w.Close()
		}
	default:
		// 纯文本，不用multipart
		topCT = `text/plain; charset="utf-8"`
		core = func() error {
			qp := quotedprintable.NewWriter(buf)
			if _, err := qp.Write([]byte(text)); err != nil {
				return err
			}
			return qp.Close()
		}
	}

	var toHeaders []string
	for _, to := range tos {
		toHeaders = append(toHeaders, (&mail.Address{Address: to}).String())
	}
	hdr := textproto.MIMEHeader{}
	if fromName != "" {
		hdr.Set("From", (&mail.Address{Name: fromName, Address: from}).String())
	} else {
		hdr.Set("From", (&mail.Address{Address: from}).String())
	}
	hdr.Set("To", strings.Join(toHeaders, ", "))
	hdr.Set("Subject", mime.QEncoding.Encode("utf-8", subject))
	hdr.Set("Date", time.Now().Format(time.RFC1123Z))
	hdr.Set("Message-ID", fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), store.RandHex(8), domain))
	hdr.Set("MIME-Version", "1.0")
	hdr.Set("Content-Type", topCT)

	// 先写头区块，再写正文
	for k, v := range hdr {
		fmt.Fprintf(buf, "%s: %s\r\n", k, v[0])
	}
	buf.WriteString("\r\n")

	if err := core(); err != nil {
		return nil
	}

	// CRLF规范化：先全部归一到LF，再统一转CRLF
	out := bytes.ReplaceAll(buf.Bytes(), []byte("\r\n"), []byte("\n"))
	out = bytes.ReplaceAll(out, []byte("\n"), []byte("\r\n"))
	return out
}
