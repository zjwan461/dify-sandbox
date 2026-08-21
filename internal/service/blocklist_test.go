package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/langgenius/dify-sandbox/internal/static"
	"github.com/langgenius/dify-sandbox/internal/types"
)

// initBlocklistConfig sets the global sandbox configuration to a minimal
// in-memory value pointing at a scratch upload directory. We bypass the full
// InitConfig (which requires a working python interpreter) so the validator
// tests run on any host.
func initBlocklistConfig(t *testing.T, blocked []string) string {
	t.Helper()

	uploadDir := t.TempDir()
	t.Setenv("PYTHON_PATH", "")
	t.Setenv("BLOCKED_COMMANDS", "")

	cfg := types.DifySandboxGlobalConfigurations{
		UploadDir:       uploadDir,
		BlockedCommands: append([]string(nil), blocked...),
	}
	static.SetDifySandboxGlobalConfigurationsForTest(cfg)
	return uploadDir
}

func TestBlockedCommandSetIncludesDefaultsAndUserEntries(t *testing.T) {
	initBlocklistConfig(t, []string{"custom-tool"})

	set := blockedCommandSet()
	// Entries that MUST be present by default.
	for _, expected := range []string{"rm", "sh", "bash", "sudo", "shred"} {
		if _, ok := set[expected]; !ok {
			t.Fatalf("expected %q in default deny-list, got %#v", expected, set)
		}
	}
	// `python3` is intentionally NOT blocked (it is the supported
	// runner) — guard the rationale so a careless edit does not silently
	// start rejecting it.
	if _, ok := set["python3"]; ok {
		t.Fatalf("python3 should not be in the default deny-list")
	}
	if _, ok := set["custom-tool"]; !ok {
		t.Fatalf("expected user-configured entry to be merged in")
	}
}

func TestBlockedCommandSetIsCaseInsensitive(t *testing.T) {
	initBlocklistConfig(t, []string{"CUSTOM-TOOL"})

	set := blockedCommandSet()
	if _, ok := set["custom-tool"]; !ok {
		t.Fatalf("expected lower-cased match for user entry, got %#v", set)
	}
}

func TestContainsShellMetachar(t *testing.T) {
	cases := []struct {
		in     string
		unsafe bool
	}{
		{"hello.py", false},
		{"-c", false},
		{"--flag=value", false},
		{"hello && rm -rf /", true},
		{"hello;rm", true},
		{"hello|grep x", true},
		{"hello`whoami`", true},
		{"$(whoami)", true},
		{"hello > /etc/passwd", true},
		{"hello\nrm", true},
		{"hello\trm", true},
		{"hello'*'", true},
		{`hello"*"`, true},
		{"hello*", true},
		{"a[b]c", true},
		{"a{b,c}", true},
		{"hello?", true},
	}
	for _, tc := range cases {
		if got := containsShellMetachar(tc.in); got != tc.unsafe {
			t.Errorf("containsShellMetachar(%q) = %v, want %v", tc.in, got, tc.unsafe)
		}
	}
}

func TestValidateCommandRequestRejectsShellMetacharsInArgs(t *testing.T) {
	initBlocklistConfig(t, nil)

	_, err := validateCommandRequest("python3", []string{"hello.py;rm -rf /"}, "")
	if err == nil {
		t.Fatalf("expected shell-metachar in args to be rejected")
	}
	if !strings.Contains(err.Error(), "shell metacharacters") {
		t.Fatalf("expected shell-metachar error, got %v", err)
	}
}

func TestValidateCommandRequestRejectsBlockedCommand(t *testing.T) {
	initBlocklistConfig(t, nil)

	for _, blocked := range []string{"rm", "sh", "bash", "sudo"} {
		_, err := validateCommandRequest(blocked, []string{"-rf", "/"}, "")
		if err == nil {
			t.Fatalf("expected %q to be blocked", blocked)
		}
		if !strings.Contains(err.Error(), "deny-list") {
			t.Fatalf("expected deny-list error for %q, got %v", blocked, err)
		}
	}
}

func TestValidateCommandRequestAllowsNonBlockedCommand(t *testing.T) {
	uploadDir := initBlocklistConfig(t, nil)

	got, err := validateCommandRequest("python3", []string{"hello.py"}, "")
	if err != nil {
		t.Fatalf("expected python3 to be allowed, got %v", err)
	}
	if got != uploadDir {
		t.Fatalf("expected resolved path to equal uploadDir %q, got %q", uploadDir, got)
	}
}

func TestValidateCommandRequestMatchesBasenameOnly(t *testing.T) {
	initBlocklistConfig(t, nil)

	// Absolute paths containing a blocked name should still be blocked
	// because we use filepath.Base to look up the deny-list.
	_, err := validateCommandRequest("/usr/bin/rm", []string{"-rf", "/"}, "")
	if err == nil {
		t.Fatalf("expected /usr/bin/rm to be blocked")
	}
}

func TestValidateCommandRequestRejectsWorkDirTraversal(t *testing.T) {
	uploadDir := initBlocklistConfig(t, nil)

	cases := []string{"../etc", "subdir/../../etc", ".."}
	for _, wd := range cases {
		_, err := validateCommandRequest("python3", []string{}, wd)
		if err == nil {
			t.Fatalf("expected work_dir %q to be rejected", wd)
		}
		if !strings.Contains(err.Error(), "work_dir") {
			t.Fatalf("expected work_dir error for %q, got %v", wd, err)
		}
	}

	// Sanity: the upload directory itself is accepted when no work_dir is
	// supplied.
	got, err := validateCommandRequest("python3", []string{}, "")
	if err != nil {
		t.Fatalf("expected empty workDir to resolve to upload dir, got %v", err)
	}
	if got != uploadDir {
		t.Fatalf("expected resolved path to equal uploadDir %q, got %q", uploadDir, got)
	}
}

func TestValidateCommandRequestAcceptsUploadDirAbsolutePath(t *testing.T) {
	uploadDir := initBlocklistConfig(t, nil)

	got, err := validateCommandRequest("python3", []string{"hello.py"}, uploadDir)
	if err != nil {
		t.Fatalf("expected upload_dir as work_dir to be accepted, got %v", err)
	}
	if got != uploadDir {
		t.Fatalf("expected resolved path to equal uploadDir %q, got %q", uploadDir, got)
	}
}

func TestValidateCommandRequestRejectsAbsolutePathOutsideUploadDir(t *testing.T) {
	initBlocklistConfig(t, nil)

	_, err := validateCommandRequest("python3", []string{}, "/etc")
	if err == nil {
		t.Fatalf("expected /etc to be rejected")
	}
	if !strings.Contains(err.Error(), "absolute path outside upload_dir") {
		t.Fatalf("expected absolute-path-outside error, got %v", err)
	}
}

func TestResolveWorkDirAllowsLegitimateSubdir(t *testing.T) {
	uploadDir := initBlocklistConfig(t, nil)

	subdir := filepath.Join(uploadDir, "scripts")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	got, err := resolveWorkDir("scripts")
	if err != nil {
		t.Fatalf("expected scripts subdir to be accepted, got %v", err)
	}
	if !strings.HasPrefix(got, uploadDir) {
		t.Fatalf("expected resolved path under uploadDir %q, got %q", uploadDir, got)
	}
}
