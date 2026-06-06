package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntDefaultAndOverride(t *testing.T) {
	const key = "LETZPLAY_TEST_PORT"
	os.Unsetenv(key)
	if got := Int(key, 8090); got != 8090 {
		t.Fatalf("default: want 8090, got %d", got)
	}
	t.Setenv(key, "9000")
	if got := Int(key, 8090); got != 9000 {
		t.Fatalf("override: want 9000, got %d", got)
	}
	t.Setenv(key, "not-a-number")
	if got := Int(key, 8090); got != 8090 {
		t.Fatalf("invalid falls back to default: want 8090, got %d", got)
	}
}

func TestBoolParsing(t *testing.T) {
	const key = "LETZPLAY_TEST_BOOL"
	for _, v := range []string{"1", "true", "yes", "on"} {
		t.Setenv(key, v)
		if !Bool(key, false) {
			t.Fatalf("%q should parse true", v)
		}
	}
	for _, v := range []string{"0", "false", "no", "off"} {
		t.Setenv(key, v)
		if Bool(key, true) {
			t.Fatalf("%q should parse false", v)
		}
	}
}

func TestLoadDotEnvRespectsRealEnvAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "" +
		"# a comment\n" +
		"\n" +
		"LETZPLAY_PORT=8090\n" +
		"export LETZPLAY_ADMIN_PASSWORD=\"from-file\"\n" +
		"LETZPLAY_GUEST_PASSWORD=party\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// A value already in the real environment must win over the file.
	t.Setenv(EnvAdminPassword, "from-env")
	os.Unsetenv(EnvPort)
	os.Unsetenv(EnvGuestPassword)
	t.Cleanup(func() { os.Unsetenv(EnvPort); os.Unsetenv(EnvGuestPassword) })

	LoadDotEnv(path)

	if got := Int(EnvPort, 0); got != 8090 {
		t.Fatalf("port from file: want 8090, got %d", got)
	}
	if got := String(EnvGuestPassword, ""); got != "party" {
		t.Fatalf("guest from file: want party, got %q", got)
	}
	if got := String(EnvAdminPassword, ""); got != "from-env" {
		t.Fatalf("real env must win over file: want from-env, got %q", got)
	}
}
