package mailer

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/emersion/go-msgauth/dkim"
)

// GenerateDKIMKey 生成RSA 2048密钥对，返回(私钥PEM, DNS TXT记录值)
func GenerateDKIMKey() (privatePEM string, dnsTXT string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	privatePEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	dnsTXT = "v=DKIM1; k=rsa; p=" + base64.StdEncoding.EncodeToString(pubDER)
	return privatePEM, dnsTXT, nil
}

// DKIMDNSTXT 从私钥计算应发布到DNS的TXT记录值
func (ms *MailSettings) DKIMDNSTXT() string {
	block, _ := pem.Decode([]byte(ms.DKIMKeyPEM))
	if block == nil {
		return ""
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return ""
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return ""
	}
	return "v=DKIM1; k=rsa; p=" + base64.StdEncoding.EncodeToString(pubDER)
}

// SignDKIM 启用DKIM时给邮件加DKIM-Signature头；未启用或密钥无效则原样返回
func SignDKIM(raw []byte, ms *MailSettings) []byte {
	if !ms.DKIMEnabled || ms.DKIMKeyPEM == "" || ms.DKIMSelector == "" || ms.Domain == "" {
		return raw
	}
	block, _ := pem.Decode([]byte(ms.DKIMKeyPEM))
	if block == nil {
		return raw
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return raw
	}
	var signed bytes.Buffer
	opts := &dkim.SignOptions{
		Domain:                 ms.Domain,
		Selector:               ms.DKIMSelector,
		Signer:                 key,
		HeaderCanonicalization: dkim.CanonicalizationRelaxed,
		BodyCanonicalization:   dkim.CanonicalizationRelaxed,
	}
	if err := dkim.Sign(&signed, bytes.NewReader(raw), opts); err != nil {
		return raw
	}
	return signed.Bytes()
}
