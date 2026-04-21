package cmdutil

import (
	"github.com/spf13/cobra"
)

// func RecordTelemetry(cmd *cobra.Command, telemetry ghtelemetry.EventRecorder) {
// 	if IsTelemetryDisabled(cmd) {
// 		return
// 	}

// 	if cmd.RunE == nil {
// 		return
// 	}

// 	currentRunE := cmd.RunE
// 	cmd.RunE = func(cmd *cobra.Command, args []string) error {
// 		runErr := currentRunE(cmd, args)

// 		var flags []string
// 		cmd.Flags().Visit(func(f *pflag.Flag) {
// 			flags = append(flags, f.Name)
// 		})
// 		slices.Sort(flags)

// 		telemetry.Record(ghtelemetry.Event{
// 			Type: "command_invocation",
// 			Dimensions: map[string]string{
// 				"command": cmd.CommandPath(),
// 				"flags":   strings.Join(flags, ","),
// 			},
// 		})

// 		return runErr
// 	}
// }

// func RecordTelemetryForSubcommands(cmd *cobra.Command, telemetry ghtelemetry.EventRecorder) {
// 	for _, c := range cmd.Commands() {
// 		RecordTelemetry(c, telemetry)
// 		RecordTelemetryForSubcommands(c, telemetry)
// 	}
// }

func DisableTelemetry(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["telemetry"] = "disabled"
}

// DisableTelemetryRecursively marks the given command and all of its descendants
// as telemetry-disabled, so that no command_invocation event is recorded when
// any of them is executed.
func DisableTelemetryRecursively(cmd *cobra.Command) {
	DisableTelemetry(cmd)
	for _, c := range cmd.Commands() {
		DisableTelemetryRecursively(c)
	}
}

func IsTelemetryDisabled(cmd *cobra.Command) bool {
	return cmd.Annotations["telemetry"] == "disabled"
}
