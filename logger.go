package slackLogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type SLSeverity string

type SLOptions struct {
	WebHook string
	Channel string
	User    string
	Label   string
}

type SlackLogger struct {
	webHook        string
	channel        string
	user           string
	label          string
	error          error
	severity       SLSeverity
	ResponseBytes  []byte
	ResponseStatus int
	ResponseError  error
}

const (
	None         SLSeverity = "none"
	Notification SLSeverity = "notification"
	Info         SLSeverity = "info"
	Warning      SLSeverity = "warning"
	Error        SLSeverity = "error"
	Critical     SLSeverity = "critical"
)

// Configure - Configure the logger
func Configure(options *SLOptions) *SlackLogger {
	var slackLogger SlackLogger
	slackLogger.webHook = options.WebHook
	slackLogger.channel = options.Channel
	slackLogger.user = options.User
	slackLogger.label = options.Label
	slackLogger.error = nil
	slackLogger.severity = None
	return &slackLogger
}

// SetError - Set the error for the logger
func (sl *SlackLogger) SetError(err error) *SlackLogger {
	sl.error = err
	sl.severity = None
	return sl
}

func (sl *SlackLogger) Severity(severity SLSeverity) *SlackLogger {
	sl.severity = severity
	return sl
}

// Notify - Send a simple notification with error wrapping
func (sl *SlackLogger) Notify(wrapMessage string) {
	severityString := sl.getSeverityString()
	wrap := fmt.Sprintf("%s%s", severityString, wrapMessage)
	sl.error = errors.Wrap(sl.error, wrap)
	sl.sendNotification()
}

// Notifyf - Send a notification with error wrapping and formatting
func (sl *SlackLogger) Notifyf(wrapMessage string, params ...interface{}) {
	severityString := sl.getSeverityString()
	wrap := fmt.Sprintf("%s%s", severityString, wrapMessage)
	sl.error = errors.Wrapf(sl.error, wrap, params...)
	sl.sendNotification()
}

// getSeverityString - Get the appropriate prefix for the wrap message
func (sl *SlackLogger) getSeverityString() string {
	switch sl.severity {
	case Notification:
		return "👉  "
	case Info:
		return "ℹ️  "
	case Warning:
		return "⚠️  "
	case Error:
		return "🔴  "
	case Critical:
		return "❌  "
	case None:
		return ""
	default:
		return ""
	}
}

// sendNotification - Send the notification with the computed message
func (sl *SlackLogger) sendNotification() {
	sl.ResponseBytes = make([]byte, 0)
	sl.ResponseStatus = 0
	sl.ResponseError = nil
	messageLabel := ""
	if sl.label != "" {
		messageLabel = fmt.Sprintf("*[%s]* ", sl.label)
	}
	msg := slack.Message{
		Msg: slack.Msg{
			Type:    "message",
			Channel: sl.channel,
			User:    sl.user,
			Text:    fmt.Sprintf("%s%v", messageLabel, sl.error),
		},
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logrus.Errorf("❌ [SlackLogger:Notify:1] [%v]", err)
		return
	}
	body := bytes.NewBuffer(msgBytes)
	request, err := http.NewRequest("POST", sl.webHook, body)
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
