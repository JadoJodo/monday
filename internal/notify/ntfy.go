package notify

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/JadoJodo/rundown/internal/config"
)

type ntfy struct{ client *http.Client }

// Ntfy returns the ntfy notifier. It POSTs to {server}/{topic}; the server URL
// comes from config, so tests point it at an httptest server with no hooks.
func Ntfy() Notifier { return ntfy{client: http.DefaultClient} }

func (ntfy) Name() string                   { return "ntfy" }
func (ntfy) Enabled(cfg config.Config) bool { return cfg.Notify.Ntfy.Enabled }

func (n ntfy) Send(ctx context.Context, cfg config.Config, msg Message) error {
	c := cfg.Notify.Ntfy
	url := strings.TrimRight(c.Server, "/") + "/" + c.Topic

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(msg.Body))
	if err != nil {
		return err
	}
	req.Header.Set("Title", msg.Title)

	priority := c.Priority
	if msg.Failed {
		// Surface failures: bump an unset/default priority and tag the message.
		if priority == "" || priority == "default" {
			priority = "high"
		}
		req.Header.Set("Tags", "warning")
	}
	if priority != "" {
		req.Header.Set("Priority", priority)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s: status %d", url, resp.StatusCode)
	}
	return nil
}
