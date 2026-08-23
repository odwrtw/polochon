package webhook

import "testing"

func TestInit(t *testing.T) {
	w := &WebHook{}
	params := []byte(`
name: webhook
hooks:
- url: https://example.com/hook
`)

	if err := w.Init(params, nil); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if len(w.hooks) != 1 {
		t.Fatalf("got %d hooks, want 1", len(w.hooks))
	}
	if got := w.hooks[0].URL; got != "https://example.com/hook" {
		t.Fatalf("got URL %q, want %q", got, "https://example.com/hook")
	}
}
