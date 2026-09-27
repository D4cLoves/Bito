package mailer

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"bito/internal/config"
)

type Mailer struct {
	host string
	port string
	from string
}

func New(cfg *config.Config) *Mailer {
	return &Mailer{
		host: cfg.SMTPHost,
		port: cfg.SMTPPort,
		from: cfg.SMTPFrom,
	}
}

func (m *Mailer) SendVerificationCode(ctx context.Context, toEmail, code string) error {
	subject := "Код подтверждения регистрации в Bito"
	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px; border: 1px solid #e0e0e0; border-radius: 8px;">
			<h2 style="color: #2c3e50; margin-bottom: 16px;">Добро пожаловать в Bito! 🃏</h2>
			<p style="color: #4a5568; font-size: 15px; line-height: 1.5;">
				Для завершения регистрации и входа в игру используйте одноразовый проверочный код:
			</p>
			<div style="background-color: #f7fafc; border: 1px dashed #cbd5e0; padding: 16px; text-align: center; border-radius: 6px; margin: 24px 0;">
				<span style="font-size: 32px; font-weight: bold; letter-spacing: 6px; color: #3182ce;">%s</span>
			</div>
			<p style="color: #718096; font-size: 13px;">
				Код действителен в течение 15 минут. Если вы не регистрировались в игре, просто проигнорируйте это письмо.
			</p>
		</div>
	`, code)

	return m.SendEmail(ctx, toEmail, subject, htmlBody)
}

func (m *Mailer) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	addr := net.JoinHostPort(m.host, m.port)

	// Формируем MIME заголовки и тело письма
	header := make(map[string]string)
	header["From"] = m.from
	header["To"] = to
	header["Subject"] = "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"
	header["Date"] = time.Now().Format(time.RFC1123Z)

	var messageBuilder strings.Builder
	for k, v := range header {
		messageBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	messageBuilder.WriteString("\r\n")
	messageBuilder.WriteString(htmlBody)

	msg := []byte(messageBuilder.String())

	// Поддержка context таймаута для безопасной отправки
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mailer: connect to smtp %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: create smtp client: %w", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if err = client.Mail(m.from); err != nil {
		return fmt.Errorf("mailer: mail from %s: %w", m.from, err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: rcpt to %s: %w", to, err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: open data stream: %w", err)
	}

	if _, err = writer.Write(msg); err != nil {
		_ = writer.Close()
		return fmt.Errorf("mailer: write message: %w", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("mailer: close data stream: %w", err)
	}

	return client.Quit()
}
