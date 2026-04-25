package certificates

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"encore.app/learning/certificates/certutil"
)

// Encore loads these from .secrets.local.cue (local) or encore secret set (prod).
var secrets struct {
	MailServer   string
	MailPort     string
	MailUsername string
	MailPassword string
	MailFrom     string
	AppURL       string
}

// ────────────────────────────────────────────────────
// Data types (mirrors send_html_email.go)
// ────────────────────────────────────────────────────

type certMailRow struct {
	DaysLeft     string
	ExpiresAt    string
	EmployeeName string
	EmployeeLink string
	Email        string
	CertName     string
	CertLink     string
}

type certMailData struct {
	Title string
	Rows  []certMailRow
}

// empInfo holds employee fields needed for the email.
type empInfo struct {
	Name  string
	Email string
}

// ────────────────────────────────────────────────────
// HTML template (same design as send_html_email.go)
// ────────────────────────────────────────────────────

const certMailTemplate = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>{{.Title}}</title>
</head>
<body style="margin:0;padding:24px;background:#f5f6f8;font-family:Arial,Helvetica,sans-serif;color:#1f2937;">

  <table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="max-width:720px;margin:0 auto;background:#ffffff;border:1px solid #e5e7eb;border-radius:8px;">

    <!-- Header -->
    <tr>
      <td style="padding:24px;text-align:center;">
        <div style="font-size:12px;color:#6b7280;margin-bottom:8px;">SANA LMS</div>
        <h1 style="margin:0;font-size:22px;font-weight:600;color:#111827;">
          {{.Title}}
        </h1>
      </td>
    </tr>

    <!-- Intro -->
    <tr>
      <td style="padding:8px 24px 20px 24px;font-size:14px;line-height:1.6;color:#6b7280;text-align:left;">
        У сотрудников истекают сертификаты в ближайшие 6 месяцев. Рекомендуем заранее запланировать продление.
      </td>
    </tr>

    <!-- Table -->
    <tr>
      <td style="padding:0 24px 24px 24px;">
        <table width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse;">
          <thead>
            <tr>
              <th align="left" style="padding:8px;border-bottom:1px solid #e5e7eb;font-size:12px;">До истечения</th>
              <th align="left" style="padding:8px;border-bottom:1px solid #e5e7eb;font-size:12px;">Дата истечения</th>
              <th align="left" style="padding:8px;border-bottom:1px solid #e5e7eb;font-size:12px;">Сотрудник</th>
              <th align="left" style="padding:8px;border-bottom:1px solid #e5e7eb;font-size:12px;">Почта</th>
              <th align="left" style="padding:8px;border-bottom:1px solid #e5e7eb;font-size:12px;">Сертификат</th>
            </tr>
          </thead>
          <tbody>
            {{range .Rows}}
            <tr>
              <td style="padding:8px;font-size:14px;color:#d97706;font-weight:500;">{{.DaysLeft}}</td>
              <td style="padding:8px;font-size:14px;">{{.ExpiresAt}}</td>
              <td style="padding:8px;font-size:14px;">
                <a href="{{.EmployeeLink}}" style="color:#2563eb;text-decoration:none;">
                  {{.EmployeeName}}
                </a>
              </td>
              <td style="padding:8px;font-size:14px;">{{.Email}}</td>
              <td style="padding:8px;font-size:14px;">
                <a href="{{.CertLink}}" style="color:#2563eb;text-decoration:none;">
                  {{.CertName}}
                </a>
              </td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </td>
    </tr>

    <!-- CTA -->
    <tr>
      <td style="padding:0 24px 24px 24px;text-align:center;">
        <a href="APP_URL_PLACEHOLDER/certificates" style="display:inline-block;padding:10px 16px;background:#364FC7;color:#ffffff;text-decoration:none;border-radius:6px;font-size:14px;">
          Перейти к сертификатам
        </a>
      </td>
    </tr>

    <!-- Footer -->
    <tr>
      <td style="padding:16px 24px;font-size:12px;color:#9ca3af;text-align:center;border-top:1px solid #e5e7eb;">
        Это автоматическое уведомление от Sana LMS.<br/>
        Пожалуйста, не отвечайте на него.
      </td>
    </tr>

  </table>

</body>
</html>`

// ────────────────────────────────────────────────────
// Build & send
// ────────────────────────────────────────────────────

func sendExpiryEmail(hrEmail, dzoID string, certs []certutil.Certificate, emps map[string]empInfo) error {
	host := secrets.MailServer
	port := secrets.MailPort
	user := secrets.MailUsername
	pass := secrets.MailPassword
	from := secrets.MailFrom

	if host == "" {
		host = "smtp.gmail.com"
	}
	if port == "" {
		port = "587"
	}
	if user == "" || pass == "" {
		return fmt.Errorf("MailUsername or MailPassword secret not set")
	}

	data := buildMailData(certs, emps)
	html, err := renderCertMailTemplate(data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	msg := buildHTMLMessage(from, hrEmail, data.Title, html)
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(host+":"+port, auth, from, []string{hrEmail}, msg)
}

func buildMailData(certs []certutil.Certificate, emps map[string]empInfo) certMailData {
	appURL := secrets.AppURL
	if appURL == "" {
		appURL = "http://localhost:4000"
	}
	now := time.Now()

	rows := make([]certMailRow, 0, len(certs))
	for _, c := range certs {
		emp := emps[c.EmployeeID]
		row := certMailRow{
			EmployeeName: emp.Name,
			EmployeeLink: appURL + "/employees/" + c.EmployeeID,
			Email:        emp.Email,
			CertName:     c.Title,
			CertLink:     appURL + "/certificates/" + c.ID,
		}
		if c.ExpiryDate != nil {
			row.ExpiresAt = c.ExpiryDate.Format("02.01.2006")
			d := int(c.ExpiryDate.Sub(now).Hours()/24) + 1
			row.DaysLeft = fmt.Sprintf("%d дн.", d)
		}
		rows = append(rows, row)
	}

	return certMailData{
		Title: "Истекают сертификаты сотрудников",
		Rows:  rows,
	}
}

func renderCertMailTemplate(data certMailData) (string, error) {
	appURL := secrets.AppURL
	if appURL == "" {
		appURL = "http://localhost:4000"
	}

	raw := strings.ReplaceAll(certMailTemplate, "APP_URL_PLACEHOLDER", appURL)

	tmpl, err := template.New("cert-mail").Parse(raw)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func buildHTMLMessage(from, to, subject, htmlBody string) []byte {
	headers := strings.Join([]string{
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
	}, "\r\n")
	return []byte(headers + "\r\n\r\n" + htmlBody)
}
