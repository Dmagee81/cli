package ghcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/gh"
	"github.com/cli/cli/v2/internal/gh/ghtelemetry"
	"github.com/cli/cli/v2/internal/telemetry"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_printError(t *testing.T) {
	cmd := &cobra.Command{}

	type args struct {
		err   error
		cmd   *cobra.Command
		debug bool
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
	}{
		{
			name: "generic error",
			args: args{
				err:   errors.New("the app exploded"),
				cmd:   nil,
				debug: false,
			},
			wantOut: "the app exploded\n",
		},
		{
			name: "DNS error",
			args: args{
				err: fmt.Errorf("DNS oopsie: %w", &net.DNSError{
					Name: "api.github.com",
				}),
				cmd:   nil,
				debug: false,
			},
			wantOut: `error connecting to api.github.com
check your internet connection or https://githubstatus.com
`,
		},
		{
			name: "Cobra flag error",
			args: args{
				err:   cmdutil.FlagErrorf("unknown flag --foo"),
				cmd:   cmd,
				debug: false,
			},
			wantOut: "unknown flag --foo\n\nUsage:\n\n",
		},
		{
			name: "unknown Cobra command error",
			args: args{
				err:   errors.New("unknown command foo"),
				cmd:   cmd,
				debug: false,
			},
			wantOut: "unknown command foo\n\nUsage:\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			printError(out, tt.args.err, tt.args.cmd, tt.args.debug)
			if gotOut := out.String(); gotOut != tt.wantOut {
				t.Errorf("printError() = %q, want %q", gotOut, tt.wantOut)
			}
		})
	}
}

func Test_newIOStreams_pager(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		config    gh.Config
		wantPager string
	}{
		{
			name: "GH_PAGER and PAGER set",
			env: map[string]string{
				"GH_PAGER": "GH_PAGER",
				"PAGER":    "PAGER",
			},
			wantPager: "GH_PAGER",
		},
		{
			name: "GH_PAGER and config pager set",
			env: map[string]string{
				"GH_PAGER": "GH_PAGER",
			},
			config:    pagerConfig(),
			wantPager: "GH_PAGER",
		},
		{
			name: "config pager and PAGER set",
			env: map[string]string{
				"PAGER": "PAGER",
			},
			config:    pagerConfig(),
			wantPager: "CONFIG_PAGER",
		},
		{
			name: "only PAGER set",
			env: map[string]string{
				"PAGER": "PAGER",
			},
			wantPager: "PAGER",
		},
		{
			name: "GH_PAGER set to blank string",
			env: map[string]string{
				"GH_PAGER": "",
				"PAGER":    "PAGER",
			},
			wantPager: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != nil {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			}
			var cfg gh.Config
			if tt.config != nil {
				cfg = tt.config
			} else {
				cfg = config.NewBlankConfig()
			}
			io := newIOStreams(cfg)
			assert.Equal(t, tt.wantPager, io.GetPager())
		})
	}
}

func Test_newIOStreams_prompt(t *testing.T) {
	tests := []struct {
		name           string
		config         gh.Config
		promptDisabled bool
		env            map[string]string
	}{
		{
			name:           "default config",
			promptDisabled: false,
		},
		{
			name:           "config with prompt disabled",
			config:         disablePromptConfig(),
			promptDisabled: true,
		},
		{
			name:           "prompt disabled via GH_PROMPT_DISABLED env var",
			env:            map[string]string{"GH_PROMPT_DISABLED": "1"},
			promptDisabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != nil {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			}
			var cfg gh.Config
			if tt.config != nil {
				cfg = tt.config
			} else {
				cfg = config.NewBlankConfig()
			}
			io := newIOStreams(cfg)
			assert.Equal(t, tt.promptDisabled, io.GetNeverPrompt())
		})
	}
}

func Test_newIOStreams_spinnerDisabled(t *testing.T) {
	tests := []struct {
		name            string
		config          gh.Config
		spinnerDisabled bool
		env             map[string]string
	}{
		{
			name:            "default config",
			spinnerDisabled: false,
		},
		{
			name:            "config with spinner disabled",
			config:          disableSpinnersConfig(),
			spinnerDisabled: true,
		},
		{
			name:            "config with spinner enabled",
			config:          enableSpinnersConfig(),
			spinnerDisabled: false,
		},
		{
			name:            "spinner disabled via GH_SPINNER_DISABLED env var = 0",
			env:             map[string]string{"GH_SPINNER_DISABLED": "0"},
			spinnerDisabled: false,
		},
		{
			name:            "spinner disabled via GH_SPINNER_DISABLED env var = false",
			env:             map[string]string{"GH_SPINNER_DISABLED": "false"},
			spinnerDisabled: false,
		},
		{
			name:            "spinner disabled via GH_SPINNER_DISABLED env var = no",
			env:             map[string]string{"GH_SPINNER_DISABLED": "no"},
			spinnerDisabled: false,
		},
		{
			name:            "spinner enabled via GH_SPINNER_DISABLED env var = 1",
			env:             map[string]string{"GH_SPINNER_DISABLED": "1"},
			spinnerDisabled: true,
		},
		{
			name:            "spinner enabled via GH_SPINNER_DISABLED env var = true",
			env:             map[string]string{"GH_SPINNER_DISABLED": "true"},
			spinnerDisabled: true,
		},
		{
			name:            "config enabled but env disabled, respects env",
			config:          enableSpinnersConfig(),
			env:             map[string]string{"GH_SPINNER_DISABLED": "true"},
			spinnerDisabled: true,
		},
		{
			name:            "config disabled but env enabled, respects env",
			config:          disableSpinnersConfig(),
			env:             map[string]string{"GH_SPINNER_DISABLED": "false"},
			spinnerDisabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var cfg gh.Config
			if tt.config != nil {
				cfg = tt.config
			} else {
				cfg = config.NewBlankConfig()
			}
			io := newIOStreams(cfg)
			assert.Equal(t, tt.spinnerDisabled, io.GetSpinnerDisabled())
		})
	}
}

func Test_newIOStreams_accessiblePrompterEnabled(t *testing.T) {
	tests := []struct {
		name                      string
		config                    gh.Config
		accessiblePrompterEnabled bool
		env                       map[string]string
	}{
		{
			name:                      "default config",
			accessiblePrompterEnabled: false,
		},
		{
			name:                      "config with accessible prompter enabled",
			config:                    enableAccessiblePrompterConfig(),
			accessiblePrompterEnabled: true,
		},
		{
			name:                      "config with accessible prompter disabled",
			config:                    disableAccessiblePrompterConfig(),
			accessiblePrompterEnabled: false,
		},
		{
			name:                      "accessible prompter enabled via GH_ACCESSIBLE_PROMPTER env var = 1",
			env:                       map[string]string{"GH_ACCESSIBLE_PROMPTER": "1"},
			accessiblePrompterEnabled: true,
		},
		{
			name:                      "accessible prompter enabled via GH_ACCESSIBLE_PROMPTER env var = true",
			env:                       map[string]string{"GH_ACCESSIBLE_PROMPTER": "true"},
			accessiblePrompterEnabled: true,
		},
		{
			name:                      "accessible prompter disabled via GH_ACCESSIBLE_PROMPTER env var = 0",
			env:                       map[string]string{"GH_ACCESSIBLE_PROMPTER": "0"},
			accessiblePrompterEnabled: false,
		},
		{
			name:                      "config disabled but env enabled, respects env",
			config:                    disableAccessiblePrompterConfig(),
			env:                       map[string]string{"GH_ACCESSIBLE_PROMPTER": "true"},
			accessiblePrompterEnabled: true,
		},
		{
			name:                      "config enabled but env disabled, respects env",
			config:                    enableAccessiblePrompterConfig(),
			env:                       map[string]string{"GH_ACCESSIBLE_PROMPTER": "false"},
			accessiblePrompterEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var cfg gh.Config
			if tt.config != nil {
				cfg = tt.config
			} else {
				cfg = config.NewBlankConfig()
			}
			io := newIOStreams(cfg)
			assert.Equal(t, tt.accessiblePrompterEnabled, io.AccessiblePrompterEnabled())
		})
	}
}

func Test_newIOStreams_colorLabels(t *testing.T) {
	tests := []struct {
		name               string
		config             gh.Config
		colorLabelsEnabled bool
		env                map[string]string
	}{
		{
			name:               "default config",
			colorLabelsEnabled: false,
		},
		{
			name:               "config with colorLabels enabled",
			config:             enableColorLabelsConfig(),
			colorLabelsEnabled: true,
		},
		{
			name:               "config with colorLabels disabled",
			config:             disableColorLabelsConfig(),
			colorLabelsEnabled: false,
		},
		{
			name:               "colorLabels enabled via `1` in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": "1"},
			colorLabelsEnabled: true,
		},
		{
			name:               "colorLabels enabled via `true` in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": "true"},
			colorLabelsEnabled: true,
		},
		{
			name:               "colorLabels enabled via `yes` in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": "yes"},
			colorLabelsEnabled: true,
		},
		{
			name:               "colorLabels disable via empty string in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": ""},
			colorLabelsEnabled: false,
		},
		{
			name:               "colorLabels disabled via `0` in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": "0"},
			colorLabelsEnabled: false,
		},
		{
			name:               "colorLabels disabled via `false` in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": "false"},
			colorLabelsEnabled: false,
		},
		{
			name:               "colorLabels disabled via `no` in GH_COLOR_LABELS env var",
			env:                map[string]string{"GH_COLOR_LABELS": "no"},
			colorLabelsEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != nil {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			}
			var cfg gh.Config
			if tt.config != nil {
				cfg = tt.config
			} else {
				cfg = config.NewBlankConfig()
			}
			io := newIOStreams(cfg)
			assert.Equal(t, tt.colorLabelsEnabled, io.ColorLabels())
		})
	}
}

func Test_mightBeGHESUser(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		config gh.Config
		want   bool
	}{
		{
			name:   "GH_ENTERPRISE_TOKEN set",
			env:    map[string]string{"GH_ENTERPRISE_TOKEN": "some-token"},
			config: config.NewBlankConfig(),
			want:   true,
		},
		{
			name:   "GITHUB_ENTERPRISE_TOKEN set",
			env:    map[string]string{"GITHUB_ENTERPRISE_TOKEN": "some-token"},
			config: config.NewBlankConfig(),
			want:   true,
		},
		{
			name:   "no env vars, config has enterprise host",
			config: config.NewFromString("hosts:\n  ghes.example.com:\n    oauth_token: abc123\n"),
			want:   true,
		},
		{
			name:   "no env vars, config has only github.com",
			config: config.NewFromString("hosts:\n  github.com:\n    oauth_token: abc123\n"),
			want:   false,
		},
		{
			name:   "no env vars, config has no hosts",
			config: config.NewBlankConfig(),
			want:   false,
		},
		{
			name:   "no env vars, config has github.com and enterprise host",
			config: config.NewFromString("hosts:\n  github.com:\n    oauth_token: abc123\n  ghes.example.com:\n    oauth_token: def456\n"),
			want:   true,
		},
		{
			name:   "no env vars, config has tenancy host",
			config: config.NewFromString("hosts:\n  my-company.ghe.com:\n    oauth_token: abc123\n"),
			want:   false,
		},
		{
			name:   "GH_HOST set to enterprise host",
			env:    map[string]string{"GH_HOST": "ghes.example.com"},
			config: config.NewBlankConfig(),
			want:   true,
		},
		{
			name:   "GH_HOST set to github.com",
			env:    map[string]string{"GH_HOST": "github.com"},
			config: config.NewBlankConfig(),
			want:   false,
		},
		{
			name:   "GH_HOST set to tenancy host",
			env:    map[string]string{"GH_HOST": "my-company.ghe.com"},
			config: config.NewBlankConfig(),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got := mightBeGHESUser(tt.config)
			assert.Equal(t, tt.want, got)
		})
	}
}

func pagerConfig() gh.Config {
	return config.NewFromString("pager: CONFIG_PAGER")
}

func disablePromptConfig() gh.Config {
	return config.NewFromString("prompt: disabled")
}

func enableAccessiblePrompterConfig() gh.Config {
	return config.NewFromString("accessible_prompter: enabled")
}

func disableAccessiblePrompterConfig() gh.Config {
	return config.NewFromString("accessible_prompter: disabled")
}

func disableSpinnersConfig() gh.Config {
	return config.NewFromString("spinner: disabled")
}

func enableSpinnersConfig() gh.Config {
	return config.NewFromString("spinner: enabled")
}

func disableColorLabelsConfig() gh.Config {
	return config.NewFromString("color_labels: disabled")
}

func enableColorLabelsConfig() gh.Config {
	return config.NewFromString("color_labels: enabled")
}

func Test_grabAllUnwrappableNestedErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "generic error",
			err:  errors.New("boom"),
			want: "*errors.errorString",
		},
		{
			name: "cmdutil sentinel",
			err:  cmdutil.CancelError,
			want: "cmdutil.cancelError",
		},
		{
			name: "single fmt wrap",
			err:  fmt.Errorf("context: %w", cmdutil.CancelError),
			want: "*fmt.wrapError,cmdutil.cancelError",
		},
		{
			name: "double fmt wrap",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", cmdutil.CancelError)),
			want: "*fmt.wrapError,*fmt.wrapError,cmdutil.cancelError",
		},
		{
			name: "terminal interrupt",
			err:  terminal.InterruptErr,
			want: "*errors.errorString",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := grabAllUnwrappableNestedErrorTypes(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_grabAllUnwrappableNestedErrorTypes_boundedAgainstCycles(t *testing.T) {
	// Build a chain deeper than the 100-element bound to ensure the loop terminates.
	err := errors.New("leaf")
	for range 101 {
		err = fmt.Errorf("wrap: %w", err)
	}
	got := grabAllUnwrappableNestedErrorTypes(err)
	parts := strings.Split(got, ",")
	assert.Len(t, parts, 100, "should be bounded to 100 entries")
}

func Test_newErrDims(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ghtelemetry.Dimensions
	}{
		{
			name: "nil error is success with no errTypes",
			err:  nil,
			want: ghtelemetry.Dimensions{"outcome": "success"},
		},
		{
			name: "generic error is failure",
			err:  errors.New("boom"),
			want: ghtelemetry.Dimensions{"outcome": "error", "errTypes": "*errors.errorString"},
		},
		{
			name: "silent error is failure",
			err:  cmdutil.SilentError,
			want: ghtelemetry.Dimensions{"outcome": "error", "errTypes": "cmdutil.silentError"},
		},
		{
			name: "cancel error is success",
			err:  cmdutil.CancelError,
			want: ghtelemetry.Dimensions{"outcome": "success", "errTypes": "cmdutil.cancelError"},
		},
		{
			name: "wrapped cancel error is success",
			err:  fmt.Errorf("wrapped: %w", cmdutil.CancelError),
			want: ghtelemetry.Dimensions{"outcome": "success", "errTypes": "*fmt.wrapError,cmdutil.cancelError"},
		},
		{
			name: "terminal interrupt is success",
			err:  terminal.InterruptErr,
			want: ghtelemetry.Dimensions{"outcome": "success", "errTypes": "*errors.errorString"},
		},
		{
			name: "wrapped terminal interrupt is success",
			err:  fmt.Errorf("wrapped: %w", terminal.InterruptErr),
			want: ghtelemetry.Dimensions{"outcome": "success", "errTypes": "*fmt.wrapError,*errors.errorString"},
		},
		{
			name: "pending error is success",
			err:  cmdutil.PendingError,
			want: ghtelemetry.Dimensions{"outcome": "success", "errTypes": "cmdutil.pendingError"},
		},
		{
			name: "no results error is success",
			err:  cmdutil.NewNoResultsError("nothing to see"),
			want: ghtelemetry.Dimensions{"outcome": "success", "errTypes": "cmdutil.NoResultsError"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, newErrDims(tt.err))
		})
	}
}

// runMainAndCaptureTelemetry runs Main() with the provided args in an isolated
// config/state directory with telemetry logging enabled, and returns the exit
// code and the telemetry payloads that were flushed to stderr.
//
// This is intentionally a janky end-to-end test: Main() reads from os.Args and
// writes to os.Stderr/os.Stdout, so we swap those process-globals around the
// call. Tests using this helper cannot run in parallel.
//
// The expectation in the long run is that we will pull this out of ghcmd.Main.
func runMainAndCaptureTelemetry(t *testing.T, args []string) (exitCode, []telemetry.SendTelemetryPayload) {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", tmpDir)
	t.Setenv("GH_DATA_DIR", tmpDir)
	t.Setenv("GH_STATE_DIR", tmpDir)

	// Enable telemetry in log mode so payloads are written to stderr rather than
	// spawned as a subprocess.
	t.Setenv("GH_PRIVATE_ENABLE_TELEMETRY", "1")
	t.Setenv("GH_TELEMETRY", "log")

	// Prevent being classified as a GHES user, which would force NoOpService.
	t.Setenv("GH_HOST", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")

	// Disable color to keep the LogFlusher output easy to parse.
	t.Setenv("NO_COLOR", "1")

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = args

	origStderr := os.Stderr
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = stderrW
	t.Cleanup(func() { os.Stderr = origStderr })

	origStdout := os.Stdout
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = stdoutW
	t.Cleanup(func() { os.Stdout = origStdout })

	// Drain stdout in the background so command output does not fill the pipe
	// buffer and block the process.
	stdoutDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, stdoutR)
		close(stdoutDone)
	}()

	exit := Main()

	require.NoError(t, stderrW.Close())
	require.NoError(t, stdoutW.Close())
	<-stdoutDone

	stderrBytes, err := io.ReadAll(stderrR)
	require.NoError(t, err)

	return exit, parseTelemetryPayloads(t, stderrBytes)
}

// parseTelemetryPayloads extracts JSON payloads written by telemetry.LogFlusher
// from the given stderr capture. LogFlusher prefixes each payload with a
// "Telemetry payload:" header followed by indented JSON.
func parseTelemetryPayloads(t *testing.T, stderr []byte) []telemetry.SendTelemetryPayload {
	t.Helper()

	const header = "Telemetry payload:"
	var payloads []telemetry.SendTelemetryPayload

	remaining := stderr
	for {
		idx := bytes.Index(remaining, []byte(header))
		if idx < 0 {
			break
		}
		remaining = remaining[idx+len(header):]

		// Locate the JSON object starting with '{' and extract until the
		// matching closing '}' at depth 0 (indented JSON emitted by
		// LogFlusher is always well-formed).
		start := bytes.IndexByte(remaining, '{')
		if start < 0 {
			break
		}
		depth := 0
		end := -1
		for i := start; i < len(remaining); i++ {
			switch remaining[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i + 1
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			break
		}

		var p telemetry.SendTelemetryPayload
		require.NoError(t, json.Unmarshal(remaining[start:end], &p))
		payloads = append(payloads, p)
		remaining = remaining[end:]
	}

	return payloads
}

func findCommandInvocation(payloads []telemetry.SendTelemetryPayload) *telemetry.PayloadEvent {
	for _, p := range payloads {
		for i := range p.Events {
			if p.Events[i].Type == "command_invocation" {
				return &p.Events[i]
			}
		}
	}
	return nil
}

func TestMain_recordsCommandInvocationTelemetry_versionSubcommand(t *testing.T) {
	exit, payloads := runMainAndCaptureTelemetry(t, []string{"gh", "version"})

	assert.Equal(t, exitOK, exit)

	event := findCommandInvocation(payloads)
	require.NotNil(t, event, "expected a command_invocation event in telemetry payloads")

	assert.Equal(t, "gh version", event.Dimensions["command"])
	// Successful runs do not populate outcome/errTypes, only error runs do.
	assert.NotContains(t, event.Dimensions, "outcome")
	assert.NotContains(t, event.Dimensions, "errTypes")
}

func TestMain_recordsCommandInvocationTelemetry_flagError(t *testing.T) {
	exit, payloads := runMainAndCaptureTelemetry(t, []string{"gh", "version", "--definitely-not-a-real-flag"})

	assert.Equal(t, exitError, exit)

	event := findCommandInvocation(payloads)
	require.NotNil(t, event, "expected a command_invocation event in telemetry payloads")

	assert.Equal(t, "gh version", event.Dimensions["command"])
	assert.Equal(t, "error", event.Dimensions["outcome"])
	assert.NotEmpty(t, event.Dimensions["errTypes"], "errTypes should be populated on error")
}
