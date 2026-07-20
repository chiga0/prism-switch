package agent

import (
	"testing"

	"github.com/chiga0/prism-switch/internal/config"
)

type mockProjector struct {
	name string
}

func (m *mockProjector) Name() string        { return m.name }
func (m *mockProjector) DisplayName() string  { return m.name }
func (m *mockProjector) ConfigPaths() []string { return nil }
func (m *mockProjector) Project(p *config.ResolvedProvider) error { return nil }
func (m *mockProjector) ReadLive() (*config.ResolvedProvider, error) { return nil, nil }

func TestRegisterAndGet(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	Register(&mockProjector{name: "alpha"})
	Register(&mockProjector{name: "beta"})

	p, err := Get("alpha")
	if err != nil {
		t.Fatalf("Get(alpha) error: %v", err)
	}
	if p.Name() != "alpha" {
		t.Errorf("Name() = %q, want alpha", p.Name())
	}

	_, err = Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestAll(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	Register(&mockProjector{name: "zeta"})
	Register(&mockProjector{name: "alpha"})
	Register(&mockProjector{name: "mid"})

	all := All()
	if len(all) != 3 {
		t.Fatalf("All() returned %d, want 3", len(all))
	}
	// Should be sorted
	if all[0].Name() != "alpha" || all[1].Name() != "mid" || all[2].Name() != "zeta" {
		t.Errorf("All() not sorted: %v, %v, %v", all[0].Name(), all[1].Name(), all[2].Name())
	}
}

func TestAvailableNames(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	Register(&mockProjector{name: "b"})
	Register(&mockProjector{name: "a"})

	names := AvailableNames()
	if names != "a, b" {
		t.Errorf("AvailableNames() = %q, want %q", names, "a, b")
	}
}

func TestAvailableNamesEmpty(t *testing.T) {
	ResetRegistry()
	defer ResetRegistry()

	if names := AvailableNames(); names != "" {
		t.Errorf("AvailableNames() = %q, want empty", names)
	}
}
