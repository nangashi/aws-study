package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/comail/colog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	SECRET_NAME   = "SlackSecret"
	SECRET_REGION = "ap-northeast-1"
)

func main() {
	colog.Register()
	colog.SetFlags(log.LstdFlags | log.Lshortfile)
	colog.ParseFields(true)

	log.Println("info: start lambda_slack_bot.")
	lambda.Start(handler)
}

func handler(context context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	secrets, err := GetSecrets()
	if err != nil {
		return respond(http.StatusInternalServerError, err)
	}

	// SlackAPIのアクセスであるか検証
	if statusCode, err := verifySlackRequest(request, secrets.SigningSecret); err != nil {
		return respond(statusCode, err)
	}
	log.Println("debug: verification ok.")
	// パラメータ解析
	event, err := parseEvent(request.Body)
	if err != nil {
		return respond(http.StatusInternalServerError, err)
	}

	api := slack.New(secrets.Token)

	switch event := event.(type) {
	case slackevents.ChallengeResponse:
		return respondWithBody(http.StatusOK, event.Challenge, nil)
	case slackevents.EventsAPIEvent:
		switch event.Type {
		case slackevents.CallbackEvent:
			switch e := event.InnerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				message := slack.MsgOptionBlocks(
					slack.NewSectionBlock(
						slack.NewTextBlockObject(slack.MarkdownType, "*hogee*\naiueo", false, false),
						nil,
						nil,
					),
					slack.NewActionBlock("A",
						slack.NewButtonBlockElement("B", "val",
							slack.NewTextBlockObject(slack.PlainTextType, "FFFF", false, false),
						).WithStyle(slack.StylePrimary),
						slack.NewButtonBlockElement("C", "val2",
							slack.NewTextBlockObject(slack.PlainTextType, "EEE", false, false),
						).WithStyle(slack.StyleDanger),
					),
				)
				if _, _, err := api.PostMessage(e.Channel, slack.MsgOptionText("message from bot!", false), message); err != nil {
					log.Println("error: ", err)
				}
			}
		}
		return respond(http.StatusOK, nil)
	case slack.InteractionCallback:
		action := event.ActionCallback.BlockActions[0]
		switch action.ActionID {
		case "C":
			blocks := slack.MsgOptionBlocks(
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.MarkdownType, "*hogee*\nfuuun", false, false),
					nil,
					nil,
				),
			)
			replaceOriginal := slack.MsgOptionReplaceOriginal(event.ResponseURL)
			if _, _, err := api.PostMessage(event.Channel.ID, replaceOriginal, blocks); err != nil {
				log.Println(err)
				return respond(http.StatusInternalServerError, nil)
			}
			return respond(http.StatusOK, nil)
		}
		return respond(http.StatusOK, nil)
	}

	return respond(http.StatusOK, nil)
}

func parseEvent(eventBody string) (interface{}, error) {
	// event subscriptionか、interactive messageの判定
	switch {
	case eventBody[:8] == "payload=":
		// イベントパース
		var imEvent slack.InteractionCallback
		queryStr, err := url.QueryUnescape(eventBody[8:])
		if err != nil {
			return imEvent, err
		}
		if err := json.Unmarshal([]byte(queryStr), &imEvent); err != nil {
			return imEvent, err
		}
		return imEvent, nil
	default:
		// イベントパース
		eventsAPIevent, err := slackevents.ParseEvent(json.RawMessage(eventBody), slackevents.OptionNoVerifyToken())
		if err != nil {
			return nil, err
		}
		if eventsAPIevent.Type == slackevents.URLVerification {
			// チャレンジレスポンス
			var challenge *slackevents.ChallengeResponse
			if err := json.Unmarshal([]byte(eventBody), &challenge); err != nil {
				return nil, err
			}
			return challenge, nil
		}
		return eventsAPIevent, nil
	}
}

func respond(statusCode int, err error) (events.APIGatewayProxyResponse, error) {
	if err != nil {
		log.Println("error: ", err)
	}
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
	}, err
}

func respondWithBody(statusCode int, body string, err error) (events.APIGatewayProxyResponse, error) {
	if err != nil {
		log.Println("error: ", err)
	}
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       body,
	}, err
}

func verifySlackRequest(event events.APIGatewayProxyRequest, signingSecret string) (int, error) {
	// リクエストログ
	if err := logRequest(event); err != nil {
		return http.StatusInternalServerError, err
	}
	// リクエスト検証
	headers := make(http.Header)
	for key, values := range event.MultiValueHeaders {
		for _, value := range values {
			headers.Add(key, value)
		}
	}
	sv, err := slack.NewSecretsVerifier(headers, signingSecret)
	if err != nil {
		return http.StatusBadRequest, err
	}
	if _, err := sv.Write([]byte(event.Body)); err != nil {
		return http.StatusInternalServerError, err
	}
	if err := sv.Ensure(); err != nil {
		return http.StatusUnauthorized, err
	}
	return http.StatusOK, nil
}

func logRequest(event events.APIGatewayProxyRequest) error {
	str, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Println("debug: APIGatewayProxyRequest: ", string(str))
	log.Println("debug: Request Body: ", event.Body)
	return nil
}
