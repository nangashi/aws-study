package aws

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

const (
	SLACK_SECRET_NAME   = "SlackSecret"
	SLACK_SECRET_REGION = "ap-northeast-1"
)

type SlackSecrets struct {
	Token                  string
	SigningSecret          string
	NotificationWebhookURL string
}

func GetSlackSecrets() (*SlackSecrets, error) {
	sess := session.Must(session.NewSession())
	svc := secretsmanager.New(sess, aws.NewConfig().WithRegion(SLACK_SECRET_REGION))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(SLACK_SECRET_NAME),
		VersionStage: aws.String("AWSCURRENT"),
	}
	result, err := svc.GetSecretValue(input)
	if err != nil {
		return nil, err
	}
	// log.Printf("%T型\n%[1]v\n", result)
	res := make(map[string]interface{})
	if err := json.Unmarshal([]byte(aws.StringValue(result.SecretString)), &res); err != nil {
		return nil, err
	}
	// log.Printf("%T型\n%[1]v\n", res)
	s := new(SlackSecrets)
	s.Token = res["Token"].(string)
	s.SigningSecret = res["SigningSecret"].(string)
	s.NotificationWebhookURL = res["NotificationWebhookURL"].(string)
	return s, nil
}
