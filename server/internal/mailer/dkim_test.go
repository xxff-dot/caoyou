package mailer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/emersion/go-msgauth/dkim"
)

func TestDKIMSignAndVerify(t *testing.T) {
	pemKey, dnsTXT, err := GenerateDKIMKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	if !strings.HasPrefix(dnsTXT, "v=DKIM1; k=rsa; p=") {
		t.Fatalf("bad dns txt: %q", dnsTXT)
	}
	ms := &MailSettings{
		Domain: "example.com", DKIMEnabled: true,
		DKIMSelector: "test", DKIMKeyPEM: pemKey,
	}
	raw := BuildMessage("", "a@example.com", []string{"b@qq.com"}, "dkim test", "body", "", nil, "example.com")
	signed := SignDKIM(raw, ms)
	if !bytes.Contains(signed, []byte("DKIM-Signature")) {
		t.Fatal("no DKIM-Signature header")
	}
	// 用LookupTXT注入公钥记录，走真实校验流程
	verifs, err := dkim.VerifyWithOptions(bytes.NewReader(signed), &dkim.VerifyOptions{
		LookupTXT: func(domain string) ([]string, error) {
			return []string{dnsTXT}, nil
		},
	})
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if len(verifs) != 1 || verifs[0].Err != nil {
		t.Fatalf("bad verification: %+v", verifs)
	}
}

func TestDKIMDisabledPassthrough(t *testing.T) {
	raw := BuildMessage("", "a@example.com", []string{"b@qq.com"}, "s", "b", "", nil, "example.com")
	if got := SignDKIM(raw, &MailSettings{}); !bytes.Equal(got, raw) {
		t.Fatal("disabled DKIM must return raw unchanged")
	}
}
