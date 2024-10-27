package main

import (
	"fmt"
	"os"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

func sendMail(to, recipientName, subject, body string) error {
	apiKey := os.Getenv("EXPENSER_MAILJET_API_KEY")
	apiSecret := os.Getenv("EXPENSER_MAILJET_API_SECRET")
	mj := mailjet.NewMailjetClient(apiKey, apiSecret)

	messagesInfo := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: "deverett@gmail.com",
				Name:  "Daniel Everett",
			},
			To: &mailjet.RecipientsV31{
				{
					Email: to,
					Name:  recipientName,
				},
			},
			Subject:  subject,
			TextPart: body,
		},
	}

	messages := mailjet.MessagesV31{Info: messagesInfo}

	_, err := mj.SendMailV31(&messages)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
