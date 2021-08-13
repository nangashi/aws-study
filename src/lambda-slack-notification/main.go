package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"project1/src/common/aws"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/comail/colog"
	"github.com/slack-go/slack"
)

type ConsoleLoginEvent struct {
	UserIdentity struct {
		UserName string `json:"userName"`
		Type     string `json:"type"`
	} `json:"userIdentity"`
	EventTime        time.Time `json:"eventTime"`
	EventName        string    `json:"eventName"`
	ResponseElements struct {
		ConsoleLogin string `json:"ConsoleLogin"`
	} `json:"responseElements"`
	AdditionalEventData struct {
		MFAUsed string `json:"MFAUsed"`
	} `json:"additionalEventData"`
	EventType       string `json:"eventType"`
	SourceIPAddress string `json:"sourceIPAddress"`
}

func main() {
	colog.Register()
	colog.SetFlags(log.LstdFlags | log.Lshortfile)
	colog.ParseFields(true)

	log.Println("info: start lambda_slack_nofitication.")
	lambda.Start(handler)
}

func handler(context context.Context, event events.SNSEvent) error {
	if eventStr, err := json.Marshal(event); err == nil {
		log.Println("debug: ", string(eventStr))
	} else {
		log.Println("warn: SNSEvent parse error. ", err)
	}

	for _, record := range event.Records {
		var (
			cloudwatch events.CloudWatchEvent
			login      ConsoleLoginEvent
		)

		if err := json.Unmarshal([]byte(record.SNS.Message), &cloudwatch); err != nil {
			log.Println("error: cloud watch event parse error: ", err)
			return err
		}
		if err := json.Unmarshal([]byte(cloudwatch.Detail), &login); err != nil {
			log.Println("error: console login event parse error: ", err)
			return err
		}

		if login.EventName != "ConsoleLogin" {
			log.Println("info: this is not login event.")
			return nil
		}

		isMFAUsed := login.AdditionalEventData.MFAUsed == "Yes"
		isLoginSuccess := login.ResponseElements.ConsoleLogin == "Success"
		isRoot := login.UserIdentity.Type == "Root"

		level := 0
		var text string
		color := map[int]string{0: "good", 1: "warning", 2: "danger"}

		if isLoginSuccess {
			var mention string
			if isRoot {
				mention = "<!channel> "
			}
			text = fmt.Sprintf("%sAWSアカウント %s へのログインがありました。", mention, cloudwatch.AccountID)
		} else {
			text = fmt.Sprintf("<!channel> AWSアカウント %s へのログイン失敗が検知されました。", cloudwatch.AccountID)
			level = 2
		}
		if isRoot {
			text += "\nRootアカウントによるログインです。"
			level = 2
		}
		if !isMFAUsed {
			text += "\nMFAが利用されていません。"
			if level == 0 {
				level = 1
			}
		}

		secrets, err := aws.GetSlackSecrets()
		if err != nil {
			log.Println("error: failed to get secrets: ", err)
			return err
		}
		slack.PostWebhook(secrets.NotificationWebhookURL, &slack.WebhookMessage{
			Username: "AWSログイン通知",
			Text:     text,
			Attachments: []slack.Attachment{
				{
					Color: color[level],
					Fields: []slack.AttachmentField{
						{
							Title: "User",
							Value: login.UserIdentity.UserName,
							Short: true,
						},
						{
							Title: "MFA",
							Value: login.AdditionalEventData.MFAUsed,
							Short: true,
						},
						{
							Title: "Time",
							Value: login.EventTime.In(time.FixedZone("Asia/Tokyo", 9*60*60)).String(),
							Short: true,
						},
						{
							Title: "IPAddress",
							Value: login.SourceIPAddress,
							Short: true,
						},
					},
				},
			},
		})

	}

	return nil
}
