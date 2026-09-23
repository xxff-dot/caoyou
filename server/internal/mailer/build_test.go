package mailer

import (
	"bytes"
	"testing"

	"github.com/jhillyerd/enmime"
)

func TestBuildAndParseRoundtrip(t *testing.T) {
	raw := BuildMessage("张三", "zhang@localhost", []string{"li@localhost", "wang@qq.com"},
		"你好 subject", "正文内容 text body", "<b>HTML body</b>",
		[]File{{Filename: "测试附件.txt", Content: []byte("file content")}},
		"localhost")

	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if env.GetHeader("Subject") != "你好 subject" {
		t.Errorf("subject mismatch: %q", env.GetHeader("Subject"))
	}
	if env.Text != "正文内容 text body" {
		t.Errorf("text mismatch: %q", env.Text)
	}
	if env.HTML != "<b>HTML body</b>" {
		t.Errorf("html mismatch: %q", env.HTML)
	}
}
