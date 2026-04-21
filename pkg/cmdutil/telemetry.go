package cmdutil

import (
	"slices"
	"strings"

	"github.com/cli/cli/v2/internal/gh"
	"github.com/cli/cli/v2/internal/gh/ghtelemetry"
	"github.com/cli/cli/v2/internal/ghinstance"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// RecordTelemetry wraps cmd.RunE so that a command_invocation event is recorded
// after the command runs. baseRepo and config are optional (nil-safe) and are
// used to guess the target host for the guessed_host_type dimension; callers
// typically pass cmdutil.Factory.BaseRepo and cmdutil.Factory.Config.
func RecordTelemetry(
	cmd *cobra.Command,
	baseRepo func() (ghrepo.Interface, error),
	config func() (gh.Config, error),
	recorder ghtelemetry.EventRecorder,
) {
	if isTelemetryDisabled(cmd) {
		return
	}

	if cmd.RunE == nil {
		return
	}

	currentRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		runErr := currentRunE(cmd, args)

		var flags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			flags = append(flags, f.Name)
		})
		slices.Sort(flags)

		host := telemetry.GuessTargetHost(cmd, baseRepo, config)

		recorder.Record(ghtelemetry.Event{
			Type: "command_invocation",
			Dimensions: map[string]string{
				"command":           cmd.CommandPath(),
				"flags":             strings.Join(flags, ","),
				"guessed_host_type": ghinstance.CategorizeHost(host),
			},
		})

		return runErr
	}
}

// RecordTelemetryForSubcommands recursively applies RecordTelemetry to every
// subcommand of cmd.
func RecordTelemetryForSubcommands(
	cmd *cobra.Command,
	baseRepo func() (ghrepo.Interface, error),
	config func() (gh.Config, error),
	recorder ghtelemetry.EventRecorder,
) {
	for _, c := range cmd.Commands() {
		RecordTelemetry(c, baseRepo, config, recorder)
		RecordTelemetryForSubcommands(c, baseRepo, config, recorder)
	}
}

func DisableTelemetry(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["telemetry"] = "disabled"
}

func DisableTelemetryForSubcommands(cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		DisableTelemetry(c)
		DisableTelemetryForSubcommands(c)
	}
}

func isTelemetryDisabled(cmd *cobra.Command) bool {
	return cmd.Annotations["telemetry"] == "disabled"
}
