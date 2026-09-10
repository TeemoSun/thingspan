package services

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"thingspan/internal/config"
)

type EmailSender interface {
	SendEmail(subject, bodyHTML, bodyPlain string) bool
}

type SMTPEmailSender struct {
	cfg *config.Config
}

func NewEmailSender(cfg *config.Config) EmailSender {
	return &SMTPEmailSender{cfg: cfg}
}

func (s *SMTPEmailSender) SendEmail(subject, bodyHTML, bodyPlain string) bool {
	if s.cfg.SMTPHost == "" || s.cfg.SMTPUser == "" || s.cfg.MailTo == "" {
		log.Printf("SMTP 未配置，跳过邮件: %s", subject)
		return false
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	from := s.cfg.SMTPUser
	to := s.cfg.MailTo

	// Encode subject with UTF-8 base64
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))

	// Build MIME message
	boundary := fmt.Sprintf("----=_Part_%d", time.Now().UnixNano())
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	msgBuilder.WriteString("MIME-Version: 1.0\r\n")
	msgBuilder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
	msgBuilder.WriteString("\r\n")

	// Plain text part
	msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msgBuilder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msgBuilder.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	msgBuilder.WriteString(base64.StdEncoding.EncodeToString([]byte(bodyPlain)))
	msgBuilder.WriteString("\r\n")

	// HTML part
	msgBuilder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msgBuilder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msgBuilder.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	msgBuilder.WriteString(base64.StdEncoding.EncodeToString([]byte(bodyHTML)))
	msgBuilder.WriteString("\r\n")

	msgBuilder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	rawMsg := []byte(msgBuilder.String())

	var client *smtp.Client
	var err error

	tlsConfig := &tls.Config{
		ServerName: s.cfg.SMTPHost,
	}

	if s.cfg.SMTPPort == 465 {
		// SSL connection
		dialer := &net.Dialer{Timeout: 30 * time.Second}
		conn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if dialErr != nil {
			log.Printf("邮件发送失败 (SSL连接失败): %v", dialErr)
			return false
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, s.cfg.SMTPHost)
		if err != nil {
			log.Printf("邮件发送失败 (初始化客户端失败): %v", err)
			return false
		}
	} else {
		// STARTTLS or plain connection
		conn, dialErr := net.DialTimeout("tcp", addr, 30*time.Second)
		if dialErr != nil {
			log.Printf("邮件发送失败 (TCP连接失败): %v", dialErr)
			return false
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, s.cfg.SMTPHost)
		if err != nil {
			log.Printf("邮件发送失败 (初始化客户端失败): %v", err)
			return false
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			if err = client.StartTLS(tlsConfig); err != nil {
				log.Printf("邮件发送失败 (STARTTLS失败): %v", err)
				return false
			}
		}
	}
	defer client.Quit()

	if s.cfg.SMTPPassword != "" {
		auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
		if err = client.Auth(auth); err != nil {
			log.Printf("邮件发送失败 (认证失败): %v", err)
			return false
		}
	}

	if err = client.Mail(from); err != nil {
		log.Printf("邮件发送失败 (MAIL FROM失败): %v", err)
		return false
	}
	if err = client.Rcpt(to); err != nil {
		log.Printf("邮件发送失败 (RCPT TO失败): %v", err)
		return false
	}

	w, err := client.Data()
	if err != nil {
		log.Printf("邮件发送失败 (DATA命令失败): %v", err)
		return false
	}
	defer w.Close()

	if _, err = w.Write(rawMsg); err != nil {
		log.Printf("邮件发送失败 (写入内容失败): %v", err)
		return false
	}

	log.Printf("邮件已发送: %s", subject)
	return true
}

