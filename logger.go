package slackLogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	attachment     string
	wrap           string
	error          error
	severity       SLSeverity
	ResponseBytes  []byte
	ResponseStatus int
	ResponseError  error
}

type slackMessage struct {
	Text   string       `json:"text,omitempty"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}
type slackBlock struct {
	Type string         `json:"type,omitempty"`
	Text slackBlockText `json:"text,omitempty"`
}
type slackBlockText struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
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

func (sl *SlackLogger) SetMessage(message string) *SlackLogger {
	sl.error = errors.New(message)
	sl.severity = None
	return sl
}

func (sl *SlackLogger) Severity(severity SLSeverity) *SlackLogger {
	sl.severity = severity
	return sl
}

// Notify - Send a simple notification with error wrapping
func (sl *SlackLogger) Notify(wrapMessage string) {
	sl.wrap = wrapMessage
	sl.sendNotification()
}

// Notifyf - Send a notification with error wrapping and formatting
func (sl *SlackLogger) Notifyf(wrapMessage string, params ...interface{}) {
	sl.error = errors.Wrapf(sl.error, wrapMessage, params...)
	sl.wrap = wrapMessage
	sl.sendNotification()
}

func (sl *SlackLogger) SetAttachment(attachment string) {
	sl.attachment = attachment
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
	var msg slackMessage
	msg.Blocks = make([]slackBlock, 0)
	//Add the message label
	msg.Blocks = append(msg.Blocks,
		slackBlock{
			Type: "section",
			Text: slackBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Location*\n%s\n\n", messageLabel),
			},
		},
		slackBlock{Type: "divider"},
		slackBlock{
			Type: "section",
			Text: slackBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Severity*\n%s\n\n", sl.getSeverityString()),
			},
		},
		slackBlock{Type: "divider"},
		slackBlock{
			Type: "section",
			Text: slackBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Message*\n%s\n\n", sl.wrap),
			},
		},
		slackBlock{Type: "divider"},
		slackBlock{
			Type: "section",
			Text: slackBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Data*\n%s\n\n", sl.error.Error()),
			},
		})

	if sl.attachment != "" {
		msg.Blocks = append(msg.Blocks,
			slackBlock{Type: "divider"},
			slackBlock{
				Type: "section",
				Text: slackBlockText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Additional Data*\n%s\n\n", sl.attachment),
				},
			})
		sl.attachment = ""
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
