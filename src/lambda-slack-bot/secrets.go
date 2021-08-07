package main

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type Secrets struct {
	Token         string
	SigningSecret string
}

func GetSecrets() (*Secrets, error) {
	sess := session.Must(session.NewSession())
	svc := secretsmanager.New(sess, aws.NewConfig().WithRegion(SECRET_REGION))
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(SECRET_NAME),
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
	s := new(Secrets)
	s.Token = res["Token"].(string)
	s.SigningSecret = res["SigningSecret"].(string)
	return s, nil
}
