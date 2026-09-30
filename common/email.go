package common

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
	"slices"
	"strings"
	"time"
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

func generateMessageID() (string, error) {
	split := strings.Split(SMTPFrom, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := strings.Split(SMTPFrom, "@")[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func shouldUseSMTPLoginAuth() bool {
	if SMTPForceAuthLogin {
		return true
	}
	return isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer)
}

func getSMTPAuth() smtp.Auth {
	return AutoSMTPAuth(SMTPAccount, SMTPToken)
}

func shouldAuthenticateSMTP() bool {
	return SMTPAccount != "" && SMTPToken != ""
}

func smtpTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         SMTPServer,
		InsecureSkipVerify: SMTPInsecureSkipVerify, // #nosec G402 -- admin-controlled SMTP compatibility option.
	}
}

func newSMTPClient(addr string) (*smtp.Client, error) {
	if SMTPSSLEnabled || (SMTPPort == 465 && !SMTPStartTLSEnabled) {
		conn, err := tls.Dial("tcp", addr, smtpTLSConfig())
		if err != nil {
			return nil, err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	if SMTPStartTLSEnabled {
		startTLSSupported, _ := client.Extension("STARTTLS")
		if !startTLSSupported {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(smtpTLSConfig()); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}

func SendEmail(subject string, receiver string, content string) error {
	return SendEmailWithAttachments(subject, receiver, content, nil)
}

func SendEmailWithAttachments(subject string, receiver string, content string, attachments []EmailAttachment) error {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	id, err2 := generateMessageID()
	if err2 != nil {
		return err2
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	var mail []byte
	if len(attachments) == 0 {
		mail = []byte(fmt.Sprintf("To: %s\r\n"+
			"From: %s <%s>\r\n"+
			"Subject: %s\r\n"+
			"Date: %s\r\n"+
			"Message-ID: %s\r\n"+ // 添加 Message-ID 头
			"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
			receiver, SystemName, SMTPFrom, encodedSubject, time.Now().Format(time.RFC1123Z), id, content))
	} else {
		boundary := "new-api-" + GetRandomString(24)
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("To: %s\r\n", receiver))
		builder.WriteString(fmt.Sprintf("From: %s <%s>\r\n", SystemName, SMTPFrom))
		builder.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
		builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
		builder.WriteString(fmt.Sprintf("Message-ID: %s\r\n", id))
		builder.WriteString("MIME-Version: 1.0\r\n")
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		writeEmailBase64(&builder, []byte(content))
		builder.WriteString("\r\n")
		for _, attachment := range attachments {
			if len(attachment.Content) == 0 {
				continue
			}
			contentType := strings.TrimSpace(attachment.ContentType)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			filename := strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(attachment.Filename)
			if strings.TrimSpace(filename) == "" {
				filename = "attachment"
			}
			mediaType, contentTypeParams, err := mime.ParseMediaType(contentType)
			if err != nil {
				mediaType = "application/octet-stream"
				contentTypeParams = map[string]string{}
			}
			contentTypeParams["name"] = filename
			formattedContentType := mime.FormatMediaType(mediaType, contentTypeParams)
			if formattedContentType == "" {
				formattedContentType = "application/octet-stream"
			}
			disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
			if disposition == "" {
				disposition = "attachment"
			}
			builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			builder.WriteString(fmt.Sprintf("Content-Type: %s\r\n", formattedContentType))
			builder.WriteString("Content-Transfer-Encoding: base64\r\n")
			builder.WriteString(fmt.Sprintf("Content-Disposition: %s\r\n\r\n", disposition))
			writeEmailBase64(&builder, attachment.Content)
			builder.WriteString("\r\n")
		}
		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		mail = []byte(builder.String())
	}
	auth := getSMTPAuth()
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	client, err := newSMTPClient(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if shouldAuthenticateSMTP() {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(SMTPFrom); err != nil {
		return err
	}
	for _, receiver := range to {
		receiver = strings.TrimSpace(receiver)
		if receiver == "" {
			continue
		}
		if err = client.Rcpt(receiver); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(mail)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	err = client.Quit()
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}

func writeEmailBase64(builder *strings.Builder, data []byte) {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		builder.WriteString(encoded[:76])
		builder.WriteString("\r\n")
		encoded = encoded[76:]
	}
	if encoded != "" {
		builder.WriteString(encoded)
		builder.WriteString("\r\n")
	}
}
