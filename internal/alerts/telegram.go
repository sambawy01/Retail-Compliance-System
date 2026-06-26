// Package alerts provides the Telegram notification service for Watch Dog.
// It subscribes to vision events on the in-process bus and sends critical
// and warning alerts to management via Telegram Bot API.
package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sambawy01/Retail-Compliance-System/internal/event"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
)

// TelegramSender sends alert messages via Telegram Bot API.
type TelegramSender struct {
	botToken string
	chatID   string
	client   *http.Client
}

// New creates a TelegramSender. If botToken or chatID is empty, it operates
// in dry-run mode (logs alerts but does not send).
func New(botToken, chatID string) *TelegramSender {
	return &TelegramSender{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Send sends a text message to the configured Telegram chat.
func (t *TelegramSender) Send(ctx context.Context, text string) error {
	if t.botToken == "" || t.chatID == "" {
		slog.Info("telegram_dry_run", "chat_id", t.chatID, "text", text)
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	data := url.Values{
		"chat_id":    {t.chatID},
		"text":       {text},
		"parse_mode": {"Markdown"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("telegram: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Subscribe registers the alert handler on the event bus for critical and
// warning vision events. Info events are not sent to Telegram.
func (t *TelegramSender) Subscribe(bus *event.Bus) {
	bus.Subscribe("vision.safety.>", t.handleAlert)
	bus.Subscribe("vision.theft.>", t.handleAlert)
	bus.Subscribe("vision.access.>", t.handleAlert)
	bus.Subscribe("vision.labor.>", t.handleAlert)
	bus.Subscribe("vision.compliance.>", t.handleAlert)
	bus.Subscribe("vision.operations.>", t.handleAlert)
	bus.Subscribe("vision.inventory.>", t.handleAlert)
	bus.Subscribe("vision.security.>", t.handleAlert)
}

// handleAlert formats and sends an alert for a vision event.
func (t *TelegramSender) handleAlert(ctx context.Context, env event.Envelope) error {
	sev := vision.ResolveSeverity(env.EventType)
	if sev == vision.SeverityInfo {
		return nil // Info events don't go to Telegram
	}

	icon := "⚠️"
	if sev == vision.SeverityCritical {
		icon = "🚨"
	}

	// Parse event type into readable name
	eventName := strings.TrimPrefix(env.EventType, "vision.")
	eventName = strings.ReplaceAll(eventName, ".", " ")
	eventName = strings.ReplaceAll(eventName, "_", " ")

	text := fmt.Sprintf(
		"%s *Watch Dog Alert*\n\n"+
			"*Event:* %s\n"+
			"*Severity:* %s\n"+
			"*Location:* %s\n"+
			"*Time:* %s\n"+
			"*Camera:* %s",
		icon,
		strings.Title(eventName),
		strings.ToUpper(string(sev)),
		env.LocationID,
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		env.EventID,
	)

	return t.Send(ctx, text)
}