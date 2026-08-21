package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/langgenius/dify-sandbox/internal/static"
)

// DefaultBlockedCommands is the deny-list of command basenames that the
// /v1/sandbox/run/command endpoint refuses to execute out of the box.
//
// The list targets commands that are either destructive, capable of breaking
// out of the upload-dir sandbox, or trivially equivalent to spawning a shell
// (which would bypass every other defence). It is intentionally broad — every
// entry here must have a clear security justification. Operators can extend
// (not shrink) the deny-list via the `blocked_commands` config field or the
// BLOCKED_COMMANDS environment variable.
var DefaultBlockedCommands = []string{
	// destructive filesystem operations
	"rm", "rmdir", "mv", "dd", "mkfs", "mkfs.ext4", "mkfs.xfs", "mkfs.btrfs",
	"shred", "wipefs", "find", "xargs",
	// permission / ownership manipulation
	"chmod", "chown", "chgrp", "setfacl", "getfacl",
	// user / group management
	"useradd", "userdel", "usermod", "groupadd", "groupdel", "groupmod",
	"passwd", "chage", "gpasswd", "newgrp",
	// system control / shutdown
	"shutdown", "reboot", "halt", "poweroff", "init", "telinit",
	"systemctl", "service",
	// mount / disk operations
	"mount", "umount", "fdisk", "parted", "losetup",
	// network stack manipulation
	"iptables", "ip6tables", "nft", "ifconfig", "ip", "route", "arp",
	"tc", "ss", "netstat",
	// privilege escalation
	"sudo", "su", "doas", "pkexec",
	// remote shell / file transfer
	"ssh", "scp", "sftp", "rsync", "slogin",
	// interactive network tools
	"curl", "wget", "nc", "ncat", "netcat", "telnet",
	// shells and shell-like interpreters — equivalent to giving away full
	// control, so they are blocked regardless of arguments
	"bash", "sh", "zsh", "dash", "fish", "csh", "tcsh", "ksh", "ash",
	"busybox", "rbash",
	// scripting interpreters that could be misused to spawn shells or eval
	// arbitrary code (Python/Node are the supported runners, see below)
	"perl", "ruby", "lua", "php", "tcl", "expect",
	// scheduled tasks
	"crontab", "at", "atd", "batch", "anacron",
	// process signalling helpers that could be used to disrupt the sandbox
	"kill", "killall", "pkill", "pgrep", "fuser",
	// kernel / hardware poking
	"modprobe", "rmmod", "insmod", "lsmod", "sysctl", "kexec", "kldload",
	// package managers — would let a request mutate the sandbox image
	"apt", "apt-get", "dpkg", "yum", "dnf", "rpm", "pacman", "apk", "pip3",
	"npm", "yarn", "pnpm", "gem", "cargo",
	// NOTE: `python3` and `node` are intentionally NOT blocked — they are
	// the primary use case of this endpoint (e.g. running a previously
	// uploaded Python script). They must instead be invoked with the
	// uploaded script as an argument; arbitrary code via `python3 -c` is
	// blocked by the shell-metacharacter check below.
}

// blockedCommandSet merges the built-in deny-list with the user-configured
// `blocked_commands` and returns a set keyed by lower-cased basename. The
// built-in list is always present so user configuration can only make the
// deny-list stricter (or add new entries) — never weaker.
func blockedCommandSet() map[string]struct{} {
	configuration := static.GetDifySandboxGlobalConfigurations()
	set := make(map[string]struct{}, len(DefaultBlockedCommands)+len(configuration.BlockedCommands))
	for _, cmd := range DefaultBlockedCommands {
		set[strings.ToLower(strings.TrimSpace(cmd))] = struct{}{}
	}
	for _, cmd := range configuration.BlockedCommands {
		set[strings.ToLower(strings.TrimSpace(cmd))] = struct{}{}
	}
	return set
}

// requestError is the unified error type returned by validateCommandRequest
// and its helpers. Keeping a single type with an explicit reason makes the
// HTTP response messages easier to understand and lets callers tell apart
// "the command is on the deny-list" from "your work_dir is malformed".
type requestError struct {
	reason  string
	details string
}

func (e *requestError) Error() string {
	if e.details == "" {
		return e.reason
	}
	return e.reason + ": " + e.details
}

// errCommandBlocked is returned when a request tries to execute a command
// (or argument) that is on the deny-list.
func errCommandBlocked(name string) *requestError {
	return &requestError{
		reason:  "command is blocked by sandbox policy",
		details: fmt.Sprintf("%q is on the deny-list", name),
	}
}

// errShellMetachar is returned when the command or one of its arguments
// contains a byte that would force the kernel to invoke a shell.
func errShellMetachar(name string) *requestError {
	return &requestError{
		reason:  "command rejected because it contains shell metacharacters",
		details: fmt.Sprintf("%q would require a shell to interpret", name),
	}
}

// errWorkDir is returned when the work_dir cannot be resolved against
// upload_dir — either because it is absolute and points outside the
// sandbox, or because it tries to traverse above upload_dir.
func errWorkDir(workDir string, why string) *requestError {
	return &requestError{
		reason:  "work_dir is invalid",
		details: fmt.Sprintf("%q %s; pass either an empty string (== upload_dir) or a relative path inside it", workDir, why),
	}
}

// shellMetachars are characters that imply shell evaluation. Any of them in
// the command or any argument causes the request to be rejected so that the
// process is launched via execve and not via a shell.
var shellMetachars = []string{
	"|", "&", ";", "<", ">", "$", "(", ")", "`", "\n", "\r", "\\",
	"*", "?", "[", "]", "{", "}", "~", "!", "#", "\x00",
}

// containsShellMetachar reports whether s contains any byte that would force
// the command line to be interpreted by a shell.
func containsShellMetachar(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == unicode.ReplacementChar {
			// invalid UTF-8 — treat as suspicious
			return true
		}
		for _, m := range shellMetachars {
			if string(r) == m {
				return true
			}
		}
		// single/double quotes also indicate a shell invocation
		if r == '\'' || r == '"' {
			return true
		}
	}
	return false
}

// resolveWorkDir validates that workDir does not escape the upload
// directory. An empty string selects the upload directory itself. The
// following inputs are accepted:
//
//   - "" or "."         → upload_dir itself
//   - "<rel-sub>"       → filepath.Join(upload_dir, rel)
//   - upload_dir itself (absolute path or any path that resolves to it)
//
// Anything else — an absolute path pointing somewhere else, or a relative
// path that escapes via ".." — is rejected with an explicit error so the
// caller can fix their request instead of getting a confusing
// "blocked by sandbox policy" message.
func resolveWorkDir(workDir string) (string, error) {
	uploadDir := static.GetDifySandboxGlobalConfigurations().UploadDir
	if uploadDir == "" {
		return "", &requestError{
			reason:  "work_dir is invalid",
			details: "sandbox has no upload_dir configured",
		}
	}

	absUpload, err := filepath.Abs(uploadDir)
	if err != nil {
		return "", err
	}

	if workDir == "" || workDir == "." {
		return absUpload, nil
	}

	if filepath.IsAbs(workDir) {
		absCandidate, absErr := filepath.Abs(workDir)
		if absErr != nil {
			return "", absErr
		}
		// The upload directory itself is always accepted, even when the
		// caller spelled it out as an absolute path. Anything else is
		// rejected so the process cannot escape the sandbox by hand.
		if absCandidate == absUpload {
			return absUpload, nil
		}
		return "", errWorkDir(workDir, "is an absolute path outside upload_dir")
	}

	cleaned := filepath.Clean(workDir)
	if cleaned == "." {
		return absUpload, nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || strings.Contains(cleaned, string(filepath.Separator)+".."+string(filepath.Separator)) || strings.HasSuffix(cleaned, string(filepath.Separator)+"..") {
		return "", errWorkDir(workDir, "tries to escape upload_dir")
	}

	joined := filepath.Join(absUpload, cleaned)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absUpload, absJoined)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") || rel == ".." {
		return "", errWorkDir(workDir, "escapes upload_dir")
	}
	return absJoined, nil
}

// validateCommandRequest enforces every sandbox-side guardrail for running
// an external command:
//
//  1. The command itself and every argument must be free of shell
//     metacharacters so that execve(2) cannot accidentally invoke a shell.
//  2. The command basename must not be on the merged deny-list.
//  3. The work directory must resolve to a path inside upload_dir.
//
// It returns the resolved absolute work directory on success so the caller
// can chdir into it before launching the process.
func validateCommandRequest(command string, args []string, workDir string) (string, error) {
	if containsShellMetachar(command) {
		return "", errShellMetachar(command)
	}
	for _, a := range args {
		if containsShellMetachar(a) {
			return "", errShellMetachar(a)
		}
	}

	base := strings.ToLower(filepath.Base(command))
	if base == "" || base == "." || base == "/" || base == ".." {
		return "", errCommandBlocked(command)
	}
	if _, blocked := blockedCommandSet()[base]; blocked {
		return "", errCommandBlocked(base)
	}

	return resolveWorkDir(workDir)
}
