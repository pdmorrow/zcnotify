package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

const (
	smtpPort  uint = 25
	smtpsPort uint = 587
)

// sendEmail Send an email.
func sendEmail(to string,
	from string,
	password string,
	ssl bool,
	server string,
	subject string,
	body string) error {
	serverAndPort := strings.Split(server, ":")
	auth := smtp.PlainAuth("", from, password, serverAndPort[0])

	if len(serverAndPort) == 1 {
		// No port specified
		if ssl {
			server += ":" + string(smtpsPort)
		} else {
			server += ":" + string(smtpPort)
		}
	}

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" + body + "\r\n")
	return smtp.SendMail(server, auth, from, []string{to}, msg)
}

// SendEmail Creates a new email using ServiceEntryChange, receipients are
// specified by the emailConfig map.
func SendEmail(emailConfigs map[string]emailConfig,
	changeEntry *ServiceEntryChange) {
	for _, emailConf := range emailConfigs {
		subject := fmt.Sprintf("[ZCNOTIFY] %s %q",
			changeEntry.ChangeType.String(),
			changeEntry.Entry.Instance)
		body, err := json.MarshalIndent(*changeEntry, "", "    ")
		if err != nil {
			log.Println("marshal error:", err.Error())
			return
		}

		err = sendEmail(emailConf.To,
			emailConf.From,
			emailConf.Password,
			emailConf.Ssl,
			emailConf.Server,
			subject,
			string(body))
		if err != nil {
			log.Println("failed to send notification email:", err.Error())
		}
	}
}
