package registry

import "testing"

func TestDefaultOrderAndLookup(t *testing.T) {
	r := Default()
	want := []string{
		"softwareupdate", "mas", "brew", "npm", "pipx", "rustup", "mise",
		"custom", "cleanup", "health",
	}
	got := r.Names()
	if len(got) != len(want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("order[%d] = %q, want %q", i, got[i], n)
		}
	}

	for _, n := range want {
		if _, ok := r.Get(n); !ok {
			t.Errorf("Get(%q) not found", n)
		}
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("Get of unknown task should fail")
	}
	if len(r.All()) != len(want) {
		t.Errorf("All() len = %d, want %d", len(r.All()), len(want))
	}
}

func TestRegisterReplaceKeepsPosition(t *testing.T) {
	r := Default()
	before := r.Names()
	// Re-register npm (same name) — order must be unchanged.
	npm, _ := r.Get("npm")
	r.Register(npm)
	after := r.Names()
	if len(before) != len(after) {
		t.Errorf("re-register changed count: %v -> %v", before, after)
	}
}
