package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/langgenius/dify-sandbox/internal/core/runner"
	runner_types "github.com/langgenius/dify-sandbox/internal/core/runner/types"
	"github.com/langgenius/dify-sandbox/internal/static"
	"github.com/langgenius/dify-sandbox/internal/types"
)

// CommandOptions describes how to launch a command via the
// /v1/sandbox/run/command endpoint.
//
// Command is the binary to execute (e.g. "python3", "/usr/bin/node").
// It is resolved via exec.LookPath inside the validated work directory, so
// absolute paths and shell features are intentionally forbidden — see
// validateCommandRequest in blocklist.go for the full set of rules.
//
// Args are forwarded verbatim to exec.Command; they are NOT parsed by a
// shell. Anything that looks like a shell metacharacter is rejected.
//
// WorkDir is a path relative to the configured upload directory; the empty
// string selects upload_dir itself. The resolved directory is used as the
// process's working directory and is also where files uploaded via
// /v1/sandbox/file/upload are kept, so a typical request flow is:
//
//	POST /v1/sandbox/file/upload        (uploads hello.py)
//	POST /v1/sandbox/run/command        (executes python3 hello.py)
//
// Timeout overrides the global worker timeout for this request when
// strictly positive; otherwise WorkerTimeout from config.yaml is used.
//
// EnableNetwork mirrors the /run endpoint semantics: it can only be true
// when the sandbox is started with enable_network: true.
type CommandOptions struct {
	Command       string
	Args          []string
	WorkDir       string
	Timeout       int
	EnableNetwork bool
}

// RunCommand launches an external binary in the sandbox-upload directory
// and returns its captured stdout, stderr, exit code, and any sandbox
// error. Every request passes through validateCommandRequest first, so
// callers can rely on it being rejected up front whenever the command or
// its arguments would be unsafe.
func RunCommand(ctx context.Context, options *CommandOptions) *types.DifySandboxResponse {
	if options == nil || strings.TrimSpace(options.Command) == "" {
		return types.ErrorResponse(-400, "command is required")
	}

	// Re-validate network opt-in just like /run does, so a request that
	// slips a network-capable binary past the deny-list still cannot talk
	// to the outside world unless the operator opted in globally.
	if err := checkOptions(&runner_types.RunnerOptions{EnableNetwork: options.EnableNetwork}); err != nil {
		return types.ErrorResponse(-400, err.Error())
	}

	workDir, err := validateCommandRequest(options.Command, options.Args, options.WorkDir)
	if err != nil {
		return types.ErrorResponse(-400, err.Error())
	}

	// Ensure the work directory exists and is a directory — an upload
	// request may have raced ahead of this call and the caller might have
	// typed a typo. Failing fast with a 400 keeps the error close to the
	// call site.
	if info, statErr := os.Stat(workDir); statErr != nil {
		if os.IsNotExist(statErr) {
			return types.ErrorResponse(-404, fmt.Sprintf("work_dir %q does not exist", options.WorkDir))
		}
		return types.ErrorResponse(-500, fmt.Sprintf("failed to stat work_dir: %v", statErr))
	} else if !info.IsDir() {
		return types.ErrorResponse(-400, fmt.Sprintf("work_dir %q is not a directory", options.WorkDir))
	}

	timeout := time.Duration(options.Timeout) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(static.GetDifySandboxGlobalConfigurations().WorkerTimeout) * time.Second
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	// Resolve the executable via exec.LookPath so that PATH-based attacks
	// (e.g. a malicious ./python3 in workDir) cannot shadow a system
	// binary. LookPath also rejects bare absolute paths that fail stat —
	// but absolute paths are already caught by the shell-metachar check
	// before we get here.
	lookPath := options.Command
	if !strings.Contains(lookPath, string(os.PathSeparator)) {
		resolved, lookupErr := exec.LookPath(lookPath)
		if lookupErr != nil {
			return types.ErrorResponse(-404, fmt.Sprintf("command %q not found in PATH", options.Command))
		}
		lookPath = resolved
	}

	cmd := exec.CommandContext(ctx, lookPath, options.Args...)
	cmd.Dir = workDir

	// Start with a clean environment so leaked secrets from the host
	// (proxy credentials, AWS_*, etc.) cannot reach the subprocess. The
	// operator may opt specific variables back in via allowed_env_vars.
	cmd.Env = []string{}
	if options.EnableNetwork {
		configuration := static.GetDifySandboxGlobalConfigurations()
		if configuration.Proxy.Socks5 != "" {
			cmd.Env = append(cmd.Env,
				fmt.Sprintf("HTTPS_PROXY=%s", configuration.Proxy.Socks5),
				fmt.Sprintf("HTTP_PROXY=%s", configuration.Proxy.Socks5),
			)
		} else if configuration.Proxy.Https != "" || configuration.Proxy.Http != "" {
			if configuration.Proxy.Https != "" {
				cmd.Env = append(cmd.Env, fmt.Sprintf("HTTPS_PROXY=%s", configuration.Proxy.Https))
			}
			if configuration.Proxy.Http != "" {
				cmd.Env = append(cmd.Env, fmt.Sprintf("HTTP_PROXY=%s", configuration.Proxy.Http))
			}
		}
		if configuration.Proxy.NoProxy != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("NO_PROXY=%s", configuration.Proxy.NoProxy))
		}
	}
	for _, envVar := range static.GetDifySandboxGlobalConfigurations().AllowedEnvVars {
		if val := os.Getenv(envVar); val != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar, val))
		}
	}

	handler := runner.NewOutputCaptureRunner()
	handler.SetTimeout(timeout)
	if err := handler.CaptureOutput(ctx, cmd); err != nil {
		return types.ErrorResponse(-500, err.Error())
	}

	return types.SuccessResponse(collectRunCodeResponse(handler.Result()))
}
