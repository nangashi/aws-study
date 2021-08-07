package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/comail/colog"
)

func BeforeAll(t *testing.T) {
	colog.Register()
	colog.SetFlags(log.LstdFlags | log.Lshortfile)
	colog.SetMinLevel(colog.LInfo)
}

func TestHandler(t *testing.T) {
	var context context.Context
	tests := []struct {
		name string
		json string
	}{
		{
			name: "MFA有りログイン成功パターン",
			json: "snsevent_login_success",
		},
		{
			name: "MFA無しログイン成功パターン",
			json: "snsevent_login_nomfa",
		},
		{
			name: "ログイン失敗パターン",
			json: "snsevent_login_failure",
		},
		{
			name: "CheckMfaイベントパターン",
			json: "snsevent_mfacheck",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json", tt.json))
			if err != nil {
				t.Fatal(err)
			}
			event := events.SNSEvent{
				Records: []events.SNSEventRecord{
					{
						SNS: events.SNSEntity{
							Message: string(message),
						},
					},
				},
			}
			if err := handler(context, event); err != nil {
				t.Fatal(err)
			}
		})
	}
}
