package notify

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// EmailService handles email notifications
type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	client   *smtp.Client
	isEnabled bool
}

// NewEmailService creates a new email notification service
func NewEmailService(host, username, password, from string, port int) *EmailService {
	return &EmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		isEnabled: username != "" && password != "",
	}
}

// Disable disables the email service
func (e *EmailService) Disable() {
	e.isEnabled = false
}

// SendEmail sends an email
func (e *EmailService) SendEmail(to, subject, body string) error {
	if !e.isEnabled {
		return nil
	}

	if to == "" {
		return fmt.Errorf("recipient email is empty")
	}

	// Build email message
	msg := e.buildMessage(to, subject, body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	// Try with TLS first
	if err := e.sendWithTLS(addr, to, msg); err == nil {
		return nil
	}

	// Fallback to STARTTLS
	return e.sendWithSTARTTLS(addr, to, msg)
}

// sendWithTLS sends email using TLS
func (e *EmailService) sendWithTLS(addr, to string, msg string) error {
	// For TLS, we connect to port 465 (smtps)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)

	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	// Try to send with TLS
	tlsConfig := &tls.Config{
		ServerName: e.host,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return err
	}

	// Auth
	if err := client.Auth(auth); err != nil {
		return err
	}

	// Set sender and recipient
	if err := client.Mail(e.username); err != nil {
		return err
	}

	if err := client.Rcpt(to); err != nil {
		return err
	}

	// Send data
	wc, err := client.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	_, err = fmt.Fprint(wc, msg)
	return err
}

// sendWithSTARTTLS sends email using STARTTLS
func (e *EmailService) sendWithSTARTTLS(addr, to string, msg string) error {
	auth := smtp.PlainAuth("", e.username, e.password, e.host)

	return smtp.SendMail(
		addr,
		auth,
		e.username,
		[]string{to},
		[]byte(msg),
	)
}

// buildMessage builds the email message
func (e *EmailService) buildMessage(to, subject, body string) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", e.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.String()
}

// SendPriceChangeEmail sends a price change email
func (e *EmailService) SendPriceChangeEmail(to, productName string, oldPrice, newPrice float64, productURL string) error {
	subject := "苹果翻新价格变动提醒"
	body := e.buildPriceChangeHTML(productName, oldPrice, newPrice, productURL)

	return e.SendEmail(to, subject, body)
}

// buildPriceChangeHTML builds the HTML for price change email
func (e *EmailService) buildPriceChangeHTML(productName string, oldPrice, newPrice float64, productURL string) string {
	changeType := "上涨"
	changeColor := "#ff4444"
	if newPrice < oldPrice {
		changeType = "下降"
		changeColor = "#00cc66"
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; color: white; border-radius: 10px 10px 0 0; }
		.content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
		.product-name { font-size: 24px; font-weight: bold; margin: 20px 0; }
		.price-change { font-size: 32px; font-weight: bold; color: %s; margin: 20px 0; }
		.price-old { text-decoration: line-through; color: #999; }
		.price-new { color: #00cc66; }
		.button { display: inline-block; padding: 12px 30px; background: #0071e3; color: white; text-decoration: none; border-radius: 20px; margin-top: 20px; }
		.footer { text-align: center; color: #999; font-size: 12px; margin-top: 30px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>🍎 ApplePrice 价格变动提醒</h1>
		</div>
		<div class="content">
			<p>您订阅的产品价格发生了变动：</p>
			<div class="product-name">%s</div>
			<div class="price-change">
				<span class="price-old">¥%.2f</span>
				→
				<span class="price-new">¥%.2f</span>
				(%s)
			</div>
			%s
			<div class="footer">
				<p>本邮件由 ApplePrice 自动发送，请勿回复。</p>
				<p>%s</p>
			</div>
		</div>
	</div>
</body>
</html>`,
		changeColor,
		productName,
		oldPrice,
		newPrice,
		changeType,
		e.buildButton(productURL),
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

// buildButton builds the HTML for a call-to-action button
func (e *EmailService) buildButton(url string) string {
	if url == "" {
		return ""
	}
	return fmt.Sprintf(`<a href="%s" class="button">查看产品</a>`, url)
}

// ValidateEmail validates an email address
func (e *EmailService) ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	if parts[0] == "" || parts[1] == "" {
		return false
	}

	if !strings.Contains(parts[1], ".") {
		return false
	}

	return true
}

// IsEnabled returns whether the email service is enabled
func (e *EmailService) IsEnabled() bool {
	return e.isEnabled
}
