package slackLogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type SLOptions struct {
	WebHook string
	Channel string
	User    string
	Label   string
}

type SlackLogger struct {
	WebHook        string
	Channel        string
	User           string
	Label          string
	Error          error
	ResponseBytes  []byte
	ResponseStatus int
	ResponseError  error
}

func Configure(options *SLOptions) *SlackLogger {
	var slackLogger SlackLogger
	slackLogger.WebHook = options.WebHook
	slackLogger.Channel = options.Channel
	slackLogger.User = options.User
	slackLogger.Label = options.Label
	slackLogger.Error = nil
	return &slackLogger
}

func (sl *SlackLogger) SetError(err error) *SlackLogger {
	sl.Error = err
	return sl
}

func (sl *SlackLogger) Notify() {
	sl.ResponseBytes = make([]byte, 0)
	sl.ResponseStatus = 0
	sl.ResponseError = nil

	msg := slack.Message{
		Msg: slack.Msg{
			Type:    "message",
			Channel: sl.Channel,
			User:    sl.User,
			Text:    fmt.Sprintf("*[%s]* %v", sl.Label, sl.Error),
		},
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logrus.Errorf("❌ [SlackLogger:Notify:1] [%v]", err)
		return
	}
	body := bytes.NewBuffer(msgBytes)
	request, err := http.NewRequest("POST", sl.WebHook, body)
	if err != nil {
		sl.ResponseError = err
		logrus.Errorf("❌ [SlackLogger:Notify:2] [%v]", err)
		return
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		sl.ResponseError = err
		logrus.Errorf("❌ [SlackLogger:Notify:3] [%v]", err)
		return
	}

	sl.ResponseStatus = response.StatusCode
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		sl.ResponseError = err
		logrus.Errorf("❌ [SlackLogger:Notify:4] [%v]", err)
		return
	}
	sl.ResponseBytes = responseBody
}
