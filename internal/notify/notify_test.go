package notify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/runner"
	"github.com/JadoJodo/monday/internal/task"
)

func TestShouldNotify(t *testing.T) {
	on := config.Default()
	on.Notify.OnSuccess = true
	off := config.Default() // on_success false

	cases := []struct {
		cfg    config.Config
		failed bool
		want   bool
	}{
		{off, false, false}, // clean run, on_success off → no
		{off, true, true},   // failures always notify
		{on, false, true},   // clean run, on_success on → yes
		{on, true, true},
	}
	for i, c := range cases {
		if got := ShouldNotify(c.cfg, c.failed); got != c.want {
			t.Errorf("case %d: ShouldNotify = %v, want %v", i, got, c.want)
		}
	}
}

func TestFromSummary(t *testing.T) {
	ok := FromSummary(runner.Summary{Results: []task.Result{
		{Name: "brew", Summary: "completed"},
		{Name: "npm", Summary: "checked"},
	}})
	if ok.Title != "monday: 2 ok" {
		t.Errorf("title = %q, want 'monday: 2 ok'", ok.Title)
	}
	if ok.Failed {
		t.Error("clean run should not be Failed")
	}
	if !strings.Contains(ok.Body, "brew — completed") {
		t.Errorf("body missing brew line: %q", ok.Body)
	}

	failed := FromSummary(runner.Summary{Results: []task.Result{
		{Name: "brew", Summary: "completed"},
		{Name: "npm", Err: errors.New("npm boom")},
	}})
	if failed.Title != "monday: 1 failed" {
		t.Errorf("title = %q, want 'monday: 1 failed'", failed.Title)
	}
	if !failed.Failed {
		t.Error("run with an error should be Failed")
	}
	if !strings.Contains(failed.Body, "npm — npm boom") {
		t.Errorf("failed body should use the error text: %q", failed.Body)
	}
}

func TestMacOSUsesArgvForm(t *testing.T) {
	fake := exec.NewFake()
	cfg := config.Default()
	// Title/body carry AppleScript metacharacters that must NOT be interpolated.
	msg := Message{Title: `monday: 1 failed`, Body: `npm — "boom" \ done`}
	if err := MacOS(fake).Send(context.Background(), cfg, msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 osascript call, got %d", len(fake.Calls))
	}
	c := fake.Calls[0]
	if c.Name != "osascript" {
		t.Errorf("expected osascript, got %q", c.Name)
	}
	if c.Args[0] != "-e" {
		t.Errorf("first arg should be -e, got %q", c.Args[0])
	}
	if !strings.Contains(c.Args[1], "on run argv") {
		t.Errorf("script should use the argv form: %q", c.Args[1])
	}
	// Title and body are passed as argv items, not embedded in the script.
	if c.Args[2] != msg.Title || c.Args[3] != msg.Body {
		t.Errorf("title/body should be argv items: %v", c.Args[2:])
	}
}

func TestMacOSEnabled(t *testing.T) {
	cfg := config.Default()
	if !MacOS(exec.NewFake()).Enabled(cfg) {
		t.Error("macos should be enabled by default")
	}
	cfg.Notify.MacOS.Enabled = false
	if MacOS(exec.NewFake()).Enabled(cfg) {
		t.Error("macos should be disabled")
	}
}

func TestNtfyPostsTitleBodyPriority(t *testing.T) {
	var gotPath, gotTitle, gotPriority, gotTags, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTitle = r.Header.Get("Title")
		gotPriority = r.Header.Get("Priority")
		gotTags = r.Header.Get("Tags")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Notify.Ntfy.Enabled = true
	cfg.Notify.Ntfy.Server = srv.URL
	cfg.Notify.Ntfy.Topic = "my-topic"
	cfg.Notify.Ntfy.Priority = "default"

	msg := Message{Title: "monday: 1 failed", Body: "npm — boom", Failed: true}
	if err := Ntfy().Send(context.Background(), cfg, msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotPath != "/my-topic" {
		t.Errorf("path = %q, want /my-topic", gotPath)
	}
	if gotTitle != msg.Title {
		t.Errorf("Title header = %q", gotTitle)
	}
	if gotBody != msg.Body {
		t.Errorf("body = %q", gotBody)
	}
	// Failure bumps a default priority to high and tags the message.
	if gotPriority != "high" {
		t.Errorf("Priority = %q, want high on failure", gotPriority)
	}
	if gotTags != "warning" {
		t.Errorf("Tags = %q, want warning on failure", gotTags)
	}
}

func TestNtfyKeepsExplicitPriority(t *testing.T) {
	var gotPriority string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPriority = r.Header.Get("Priority")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Notify.Ntfy.Enabled = true
	cfg.Notify.Ntfy.Server = srv.URL
	cfg.Notify.Ntfy.Topic = "t"
	cfg.Notify.Ntfy.Priority = "min"

	if err := Ntfy().Send(context.Background(), cfg, Message{Title: "ok", Failed: false}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotPriority != "min" {
		t.Errorf("explicit priority should be kept, got %q", gotPriority)
	}
}

func TestNtfyNon2xxErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Notify.Ntfy.Enabled = true
	cfg.Notify.Ntfy.Server = srv.URL
	cfg.Notify.Ntfy.Topic = "t"

	if err := Ntfy().Send(context.Background(), cfg, Message{Title: "x"}); err == nil {
		t.Error("non-2xx response should error")
	}
}

func TestDispatchSkipsDisabledAndAggregates(t *testing.T) {
	cfg := config.Default()

	goodSent, disabledSent := false, false
	good := stubNotifier{name: "good", enabled: true, sent: &goodSent}
	bad := stubNotifier{name: "bad", enabled: true, err: errors.New("delivery failed")}
	disabled := stubNotifier{name: "disabled", enabled: false, err: errors.New("should not run"), sent: &disabledSent}

	errs := Dispatch(context.Background(), cfg, Message{Title: "t"}, good, bad, disabled)
	if len(errs) != 1 {
		t.Fatalf("expected 1 aggregated error, got %d (%v)", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "bad") {
		t.Errorf("aggregated error should name the channel: %v", errs[0])
	}
	if disabledSent {
		t.Error("disabled notifier should be skipped")
	}
	if !goodSent {
		t.Error("enabled notifier should send")
	}
}

type stubNotifier struct {
	name    string
	enabled bool
	err     error
	sent    *bool
}

func (s stubNotifier) Name() string               { return s.name }
func (s stubNotifier) Enabled(config.Config) bool { return s.enabled }
func (s stubNotifier) Send(context.Context, config.Config, Message) error {
	if s.sent != nil {
		*s.sent = true
	}
	return s.err
}
