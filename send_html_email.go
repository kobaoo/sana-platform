package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type CertificateRow struct {
	DaysLeft     string
	ExpiresAt    string
	EmployeeName string
	EmployeeLink string
	Email        string
	CertName     string
	CertLink     string
}

type EmailData struct {
	Title string
	Rows  []CertificateRow
}

const htmlTemplate = `<!doctype html>
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
  У сотрудников истекают сертификаты в ближайшие 90 дней. Рекомендуем заранее запланировать продление.
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
  <a href="https://example.com" style="display:inline-block;padding:10px 16px;background:#364FC7;color:#ffffff;text-decoration:none;border-radius:6px;font-size:14px;">
    Перейти к заявкам
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

func getEnvOrFail(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		fmt.Fprintf(os.Stderr, "missing required env variable: %s\n", name)
		os.Exit(1)
	}
	return value
}

func buildHTML(data EmailData) (string, error) {
	tmpl, err := template.New("certificate").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}

	return out.String(), nil
}

func buildMIMEMessage(from string, to []string, subject string, htmlBody string) []byte {
	headers := []string{
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
	}

	message := strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody
	return []byte(message)
}

func writePreviewFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func main() {
	subjectFlag := flag.String("subject", "Сертификаты истекают в ближайшие 90 дней", "email subject")
	toFlag := flag.String("to", "", "comma-separated recipients; overrides SMTP_TO")
	previewFlag := flag.Bool("preview", false, "build local preview files and skip SMTP send")
	htmlOutFlag := flag.String("html-out", "./tmp/certificates-preview.html", "preview html output path")
	emlOutFlag := flag.String("eml-out", "./tmp/certificates-preview.eml", "preview eml output path")
	flag.Parse()

	toCSV := strings.TrimSpace(*toFlag)
	if toCSV == "" && !*previewFlag {
		toCSV = getEnvOrFail("SMTP_TO")
	}
	if toCSV == "" && *previewFlag {
		toCSV = "preview@example.com"
	}

	var to []string
	for _, item := range strings.Split(toCSV, ",") {
		email := strings.TrimSpace(item)
		if email != "" {
			to = append(to, email)
		}
	}

	if len(to) == 0 {
		fmt.Fprintln(os.Stderr, "recipient list is empty")
		os.Exit(1)
	}

	data := EmailData{
		Title: "Истекают сертификаты сотрудников",
		Rows: []CertificateRow{
			{
				DaysLeft:     "12 дней",
				ExpiresAt:    "28.04.2026",
				EmployeeName: "Анна Смирнова",
				EmployeeLink: "https://example.com/employees/anna-smirnova",
				Email:        "anna.smirnova@example.com",
				CertName:     "Product Design Advanced",
				CertLink:     "https://example.com/certificates/product-design-advanced",
			},
			{
				DaysLeft:     "20 дней",
				ExpiresAt:    "23.04.2026",
				EmployeeName: "Иван Петров",
				EmployeeLink: "https://example.com/employees/ivan-petrov",
				Email:        "ivan.petrov@example.com",
				CertName:     "Frontend Engineer Pro",
				CertLink:     "https://example.com/certificates/frontend-engineer-pro",
			},
		},
	}

	htmlBody, err := buildHTML(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build html template: %v\n", err)
		os.Exit(1)
	}

	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = "no-reply@example.com"
	}

	if *previewFlag {
		htmlPath := filepath.Clean(*htmlOutFlag)
		if err := writePreviewFile(htmlPath, []byte(htmlBody)); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write html preview: %v\n", err)
			os.Exit(1)
		}

		emlPath := filepath.Clean(*emlOutFlag)
		emlMessage := buildMIMEMessage(from, to, *subjectFlag, htmlBody)
		if err := writePreviewFile(emlPath, emlMessage); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write eml preview: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("preview files created:\n- %s\n- %s\n", htmlPath, emlPath)
		return
	}

	host := getEnvOrFail("SMTP_HOST")
	portRaw := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if portRaw == "" {
		portRaw = "587"
	}

	if _, err := strconv.Atoi(portRaw); err != nil {
		fmt.Fprintf(os.Stderr, "SMTP_PORT must be a number, got: %s\n", portRaw)
		os.Exit(1)
	}

	user := getEnvOrFail("SMTP_USER")
	pass := getEnvOrFail("SMTP_PASS")
	from = getEnvOrFail("SMTP_FROM")

	addr := host + ":" + portRaw
	auth := smtp.PlainAuth("", user, pass, host)
	message := buildMIMEMessage(from, to, *subjectFlag, htmlBody)

	if err := smtp.SendMail(addr, auth, from, to, message); err != nil {
		fmt.Fprintf(os.Stderr, "failed to send email: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("email sent to %s\n", strings.Join(to, ", "))
}