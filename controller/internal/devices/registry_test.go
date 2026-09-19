package devices

import (
	"os"
	"path/filepath"
	"testing"
)

const fixture = `
default_paths:
  - /interfaces/interface/state/oper-status

targets:
  - name: pe1
    address: 172.20.20.11:9339
    tls:
      insecure: true
  - name: pe2
    address: 172.20.20.12:9339
    tls:
      insecure: true
    paths:
      - /components/component/state
`

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "devices.yaml")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestLoadAppliesDefaultPaths(t *testing.T) {
	reg, err := Load(writeFixture(t))
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}

	if len(reg.Targets) != 2 {
		t.Fatalf("got %d targets, want 2", len(reg.Targets))
	}

	pe1 := reg.Targets[0]
	if pe1.Name != "pe1" || pe1.Address != "172.20.20.11:9339" {
		t.Errorf("unexpected pe1: %+v", pe1)
	}
	if len(pe1.Paths) != 1 || pe1.Paths[0] != "/interfaces/interface/state/oper-status" {
		t.Errorf("pe1 should inherit default_paths, got %v", pe1.Paths)
	}
	if !pe1.TLS.Insecure {
		t.Errorf("pe1.TLS.Insecure should be true")
	}

	pe2 := reg.Targets[1]
	if len(pe2.Paths) != 1 || pe2.Paths[0] != "/components/component/state" {
		t.Errorf("pe2 should keep its own paths, got %v", pe2.Paths)
	}
}

func TestLoadRejectsMissingFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("targets:\n  - address: 1.2.3.4:9339\n"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for target missing a name")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load("/nonexistent/devices.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}
