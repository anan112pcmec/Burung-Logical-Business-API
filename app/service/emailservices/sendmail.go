package emailservices

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func Getenvi(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		fmt.Println("[Env] Found:", key, "=", value)
		return value
	}
	fmt.Println("[Env] Fallback:", key, "=", fallback)
	return fallback
}

func SendMail(to []string, cc []string, subject, message string) error {
	fmt.Println("[SendMail] Preparing email...")

	from := Getenvi("CONFIG_AUTH_EMAIL", "")
	fmt.Println("[SendMail] From:", from)

	body := "From: " + Getenvi("CONFIG_SENDER_NAME", "") + "\n" +
		"To: " + strings.Join(to, ",") + "\n" +
		"Cc: " + strings.Join(cc, ",") + "\n" +
		"Subject: " + subject + "\n\n" +
		message

	fmt.Println("[SendMail] Email body prepared")

	authEmail := Getenvi("CONFIG_AUTH_EMAIL", "")
	authPassword := Getenvi("CONFIG_AUTH_PASSWORD", "")
	smtpHost := Getenvi("CONFIG_SMTP_HOST", "")
	smtpPort := Getenvi("CONFIG_SMTP_PORT", "25")

	auth := smtp.PlainAuth("", authEmail, authPassword, smtpHost)
	smtpAddr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	fmt.Println("[SendMail] SMTP server:", smtpAddr)
	fmt.Println("[SendMail] Auth configured")
	fmt.Println("ini emailnya", authEmail, authPassword, smtpHost, smtpPort)

	recipients := append(to, cc...)
	fmt.Println("[SendMail] Recipients:", recipients)

	err := smtp.SendMail(smtpAddr, auth, from, recipients, []byte(body))
	if err != nil {
		fmt.Println("[SendMail] ERROR sending mail:", err)
		return err
	}

	fmt.Println("[SendMail] Email sent successfully")
	return nil
}
