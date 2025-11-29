// Package dispatchers provides alert dispatch implementations.
package dispatchers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
)

// SlackDispatcher dispatches alerts to Slack.
type SlackDispatcher struct {
	webhookURL string
	channel    string
	client     *http.Client
	logger     *logging.Logger
	enabled    bool
}

// SlackMessage represents a Slack message payload.
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment represents a Slack attachment.
type SlackAttachment struct {
	Color      string       `json:"color"`
	Title      string       `json:"title"`
	Text       string       `json:"text"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	MarkdownIn []string     `json:"mrkdwn_in,omitempty"`
}

// SlackField represents a Slack field.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackDispatcher creates a new SlackDispatcher.
func NewSlackDispatcher(webhookURL, channel string, logger *logging.Logger, enabled bool) *SlackDispatcher {
	return &SlackDispatcher{
		webhookURL: webhookURL,
		channel:    channel,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:  logger,
		enabled: enabled,
	}
}

// Name returns the dispatcher name.
func (d *SlackDispatcher) Name() string {
	return "slack"
}

// Enabled returns whether the dispatcher is enabled.
func (d *SlackDispatcher) Enabled() bool {
	return d.enabled && d.webhookURL != ""
}

// Dispatch sends an alert to Slack.
func (d *SlackDispatcher) Dispatch(ctx context.Context, alert *models.Alert) error {
	if !d.Enabled() {
		d.logger.Debug("slack dispatcher disabled, skipping")
		return nil
	}

	color := d.severityToColor(alert.Severity)
	emoji := d.severityToEmoji(alert.Severity)

	message := SlackMessage{
		Channel:   d.channel,
		Username:  "Alert Engine",
		IconEmoji: ":rotating_light:",
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: fmt.Sprintf("%s %s", emoji, alert.Title),
				Text:  alert.Message,
				Fields: []SlackField{
					{Title: "Service", Value: string(alert.ServiceName), Short: true},
					{Title: "Severity", Value: string(alert.Severity), Short: true},
					{Title: "Metric", Value: string(alert.MetricType), Short: true},
					{Title: "Value", Value: fmt.Sprintf("%.2f", alert.CurrentValue), Short: true},
				},
				Footer:     fmt.Sprintf("Alert ID: %s", alert.ID),
				Timestamp:  alert.Timestamp.Unix(),
				MarkdownIn: []string{"text"},
			},
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API returned status %d: %s", resp.StatusCode, string(body))
	}

	d.logger.Info("alert dispatched to slack",
		zap.String("alert_id", alert.ID),
		zap.String("channel", d.channel),
	)

	return nil
}

func (d *SlackDispatcher) severityToColor(severity models.AlertSeverity) string {
	switch severity {
	case models.AlertSeverityCritical:
		return "#dc3545" // Red
	case models.AlertSeverityWarning:
		return "#ffc107" // Yellow
	case models.AlertSeverityInfo:
		return "#17a2b8" // Blue
	default:
		return "#6c757d" // Gray
	}
}

func (d *SlackDispatcher) severityToEmoji(severity models.AlertSeverity) string {
	switch severity {
	case models.AlertSeverityCritical:
		return "🚨"
	case models.AlertSeverityWarning:
		return "⚠️"
	case models.AlertSeverityInfo:
		return "ℹ️"
	default:
		return "📢"
	}
}

// EmailDispatcher dispatches alerts via SendGrid.
type EmailDispatcher struct {
	apiKey     string
	fromEmail  string
	fromName   string
	recipients []string
	client     *http.Client
	logger     *logging.Logger
	enabled    bool
}

// SendGridPayload represents the SendGrid API payload.
type SendGridPayload struct {
	Personalizations []Personalization `json:"personalizations"`
	From             EmailAddress      `json:"from"`
	Subject          string            `json:"subject"`
	Content          []EmailContent    `json:"content"`
}

// Personalization represents SendGrid personalization.
type Personalization struct {
	To []EmailAddress `json:"to"`
}

// EmailAddress represents an email address.
type EmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// EmailContent represents email content.
type EmailContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NewEmailDispatcher creates a new EmailDispatcher.
func NewEmailDispatcher(apiKey, fromEmail, fromName string, recipients []string, logger *logging.Logger, enabled bool) *EmailDispatcher {
	return &EmailDispatcher{
		apiKey:     apiKey,
		fromEmail:  fromEmail,
		fromName:   fromName,
		recipients: recipients,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logger,
		enabled: enabled,
	}
}

// Name returns the dispatcher name.
func (d *EmailDispatcher) Name() string {
	return "email"
}

// Enabled returns whether the dispatcher is enabled.
func (d *EmailDispatcher) Enabled() bool {
	return d.enabled && d.apiKey != "" && len(d.recipients) > 0
}

// Dispatch sends an alert via email.
func (d *EmailDispatcher) Dispatch(ctx context.Context, alert *models.Alert) error {
	if !d.Enabled() {
		d.logger.Debug("email dispatcher disabled, skipping")
		return nil
	}

	tos := make([]EmailAddress, len(d.recipients))
	for i, r := range d.recipients {
		tos[i] = EmailAddress{Email: r}
	}

	subject := fmt.Sprintf("[%s] %s - %s", alert.Severity, alert.ServiceName, alert.Title)
	body := d.buildEmailBody(alert)

	payload := SendGridPayload{
		Personalizations: []Personalization{
			{To: tos},
		},
		From: EmailAddress{
			Email: d.fromEmail,
			Name:  d.fromName,
		},
		Subject: subject,
		Content: []EmailContent{
			{Type: "text/html", Value: body},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.apiKey))

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendgrid API returned status %d: %s", resp.StatusCode, string(body))
	}

	d.logger.Info("alert dispatched via email",
		zap.String("alert_id", alert.ID),
		zap.Int("recipients", len(d.recipients)),
	)

	return nil
}

func (d *EmailDispatcher) buildEmailBody(alert *models.Alert) string {
	severityColor := ""
	switch alert.Severity {
	case models.AlertSeverityCritical:
		severityColor = "#dc3545"
	case models.AlertSeverityWarning:
		severityColor = "#ffc107"
	default:
		severityColor = "#17a2b8"
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: %s; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 15px 0; }
        .detail-row { display: flex; margin-bottom: 10px; }
        .detail-label { font-weight: bold; width: 120px; }
        .footer { font-size: 12px; color: #6c757d; margin-top: 20px; padding-top: 10px; border-top: 1px solid #dee2e6; }
    </style>
</head>
<body>
    <div class="header">
        <h2>%s</h2>
    </div>
    <div class="content">
        <p>%s</p>
        <div class="details">
            <div class="detail-row"><span class="detail-label">Service:</span> %s</div>
            <div class="detail-row"><span class="detail-label">Severity:</span> %s</div>
            <div class="detail-row"><span class="detail-label">Metric:</span> %s</div>
            <div class="detail-row"><span class="detail-label">Current Value:</span> %.2f</div>
            <div class="detail-row"><span class="detail-label">Threshold:</span> %.2f</div>
            <div class="detail-row"><span class="detail-label">Timestamp:</span> %s</div>
        </div>
        <div class="footer">
            Alert ID: %s<br>
            This alert was generated by the Microservices Platform Alert Engine.
        </div>
    </div>
</body>
</html>
`,
		severityColor,
		alert.Title,
		alert.Message,
		alert.ServiceName,
		alert.Severity,
		alert.MetricType,
		alert.CurrentValue,
		alert.Threshold,
		alert.Timestamp.Format(time.RFC3339),
		alert.ID,
	)
}

// WebhookDispatcher dispatches alerts to generic webhooks.
type WebhookDispatcher struct {
	urls    []string
	headers map[string]string
	client  *http.Client
	logger  *logging.Logger
	enabled bool
}

// WebhookPayload represents the webhook payload.
type WebhookPayload struct {
	ID           string            `json:"id"`
	ServiceName  string            `json:"service_name"`
	MetricType   string            `json:"metric_type"`
	Severity     string            `json:"severity"`
	Title        string            `json:"title"`
	Message      string            `json:"message"`
	CurrentValue float64           `json:"current_value"`
	Threshold    float64           `json:"threshold"`
	Timestamp    string            `json:"timestamp"`
	Labels       map[string]string `json:"labels"`
}

// NewWebhookDispatcher creates a new WebhookDispatcher.
func NewWebhookDispatcher(urls []string, headers map[string]string, logger *logging.Logger, enabled bool) *WebhookDispatcher {
	return &WebhookDispatcher{
		urls:    urls,
		headers: headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:  logger,
		enabled: enabled,
	}
}

// Name returns the dispatcher name.
func (d *WebhookDispatcher) Name() string {
	return "webhook"
}

// Enabled returns whether the dispatcher is enabled.
func (d *WebhookDispatcher) Enabled() bool {
	return d.enabled && len(d.urls) > 0 && d.urls[0] != ""
}

// Dispatch sends an alert to all configured webhooks.
func (d *WebhookDispatcher) Dispatch(ctx context.Context, alert *models.Alert) error {
	if !d.Enabled() {
		d.logger.Debug("webhook dispatcher disabled, skipping")
		return nil
	}

	payload := WebhookPayload{
		ID:           alert.ID,
		ServiceName:  string(alert.ServiceName),
		MetricType:   string(alert.MetricType),
		Severity:     string(alert.Severity),
		Title:        alert.Title,
		Message:      alert.Message,
		CurrentValue: alert.CurrentValue,
		Threshold:    alert.Threshold,
		Timestamp:    alert.Timestamp.Format(time.RFC3339),
		Labels:       alert.Labels,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var lastErr error
	successCount := 0

	for _, url := range d.urls {
		if url == "" {
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
		if err != nil {
			d.logger.Error("failed to create webhook request",
				zap.String("url", url),
				zap.Error(err),
			)
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		for k, v := range d.headers {
			req.Header.Set(k, v)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			d.logger.Error("failed to send webhook",
				zap.String("url", url),
				zap.Error(err),
			)
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			d.logger.Error("webhook returned error status",
				zap.String("url", url),
				zap.Int("status", resp.StatusCode),
			)
			lastErr = fmt.Errorf("webhook %s returned status %d", url, resp.StatusCode)
			continue
		}

		successCount++
		d.logger.Debug("webhook sent successfully",
			zap.String("url", url),
			zap.String("alert_id", alert.ID),
		)
	}

	if successCount > 0 {
		d.logger.Info("alert dispatched via webhooks",
			zap.String("alert_id", alert.ID),
			zap.Int("success_count", successCount),
			zap.Int("total_webhooks", len(d.urls)),
		)
		return nil
	}

	return lastErr
}
