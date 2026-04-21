package cmdutil_test

import (
	"fmt"
	"testing"

	"github.com/cli/cli/v2/internal/telemetry"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordTelemetry(t *testing.T) {
	t.Run("records command path and flags", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		cmd := &cobra.Command{
			Use:  "list",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		cmd.Flags().Bool("web", false, "")
		cmd.Flags().String("repo", "", "")

		parent := &cobra.Command{Use: "pr"}
		root := &cobra.Command{Use: "gh"}
		root.AddCommand(parent)
		parent.AddCommand(cmd)

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)

		require.NoError(t, cmd.Flags().Set("web", "true"))
		require.NoError(t, cmd.Flags().Set("repo", "cli/cli"))
		require.NoError(t, cmd.RunE(cmd, nil))

		require.Len(t, recorder.Events, 1)
		event := recorder.Events[0]
		assert.Equal(t, "command_invocation", event.Type)
		assert.Equal(t, "gh pr list", event.Dimensions["command"])
		assert.Equal(t, "repo,web", event.Dimensions["flags"])
	})

	t.Run("is a no-op when original RunE is nil", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		cmd := &cobra.Command{Use: "test"}

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)

		assert.Nil(t, cmd.RunE, "RunE should remain nil when it was nil before")
		assert.Empty(t, recorder.Events, "no telemetry should be recorded")
	})

	t.Run("propagates error from original RunE", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		expectedErr := fmt.Errorf("something went wrong")
		cmd := &cobra.Command{
			Use:  "fail",
			RunE: func(cmd *cobra.Command, args []string) error { return expectedErr },
		}

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)

		err := cmd.RunE(cmd, nil)
		assert.ErrorIs(t, err, expectedErr)
		// Telemetry is still recorded even on error
		require.Len(t, recorder.Events, 1)
		assert.Equal(t, "command_invocation", recorder.Events[0].Type)
	})

	t.Run("flags are sorted alphabetically", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		cmd := &cobra.Command{
			Use:  "test",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		cmd.Flags().Bool("zebra", false, "")
		cmd.Flags().Bool("alpha", false, "")
		cmd.Flags().Bool("middle", false, "")

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)

		require.NoError(t, cmd.Flags().Set("zebra", "true"))
		require.NoError(t, cmd.Flags().Set("alpha", "true"))
		require.NoError(t, cmd.Flags().Set("middle", "true"))
		require.NoError(t, cmd.RunE(cmd, nil))

		require.Len(t, recorder.Events, 1)
		assert.Equal(t, "alpha,middle,zebra", recorder.Events[0].Dimensions["flags"])
	})

	t.Run("no flags set records empty flags string", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		cmd := &cobra.Command{
			Use:  "test",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		cmd.Flags().Bool("unused", false, "")

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)
		require.NoError(t, cmd.RunE(cmd, nil))

		require.Len(t, recorder.Events, 1)
		assert.Equal(t, "", recorder.Events[0].Dimensions["flags"])
	})

	t.Run("skips commands with telemetry disabled", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		cmd := &cobra.Command{
			Use:  "internal",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		cmdutil.DisableTelemetry(cmd)
		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)

		require.NoError(t, cmd.RunE(cmd, nil))
		assert.Empty(t, recorder.Events, "telemetry should not be recorded for disabled commands")
	})

	t.Run("records guessed_host_type from --repo flag", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		cmd := &cobra.Command{
			Use:  "view",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		cmd.Flags().String("repo", "", "")
		root := &cobra.Command{Use: "gh"}
		root.AddCommand(cmd)

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)

		require.NoError(t, cmd.Flags().Set("repo", "cli/cli"))
		require.NoError(t, cmd.RunE(cmd, nil))

		require.Len(t, recorder.Events, 1)
		assert.Equal(t, "github.com", recorder.Events[0].Dimensions["guessed_host_type"])
	})

	t.Run("records uncategorized when no signal is available", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}
		t.Setenv("GH_REPO", "")
		cmd := &cobra.Command{
			Use:  "view",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		root := &cobra.Command{Use: "gh"}
		root.AddCommand(cmd)

		cmdutil.RecordTelemetry(cmd, nil, nil, recorder)
		require.NoError(t, cmd.RunE(cmd, nil))

		require.Len(t, recorder.Events, 1)
		assert.Equal(t, "uncategorized", recorder.Events[0].Dimensions["guessed_host_type"])
	})
}

func TestRecordTelemetryForSubcommands(t *testing.T) {
	t.Run("instruments nested subcommands", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}

		root := &cobra.Command{Use: "gh"}
		parent := &cobra.Command{Use: "pr"}
		child := &cobra.Command{
			Use:  "list",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		root.AddCommand(parent)
		parent.AddCommand(child)

		cmdutil.RecordTelemetryForSubcommands(root, nil, nil, recorder)
		require.NoError(t, child.RunE(child, nil))

		require.Len(t, recorder.Events, 1)
		assert.Equal(t, "command_invocation", recorder.Events[0].Type)
		assert.Equal(t, "gh pr list", recorder.Events[0].Dimensions["command"])
	})

	t.Run("skips subcommands with nil RunE", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}

		root := &cobra.Command{Use: "gh"}
		child := &cobra.Command{Use: "help"} // no RunE
		root.AddCommand(child)

		cmdutil.RecordTelemetryForSubcommands(root, nil, nil, recorder)

		assert.Nil(t, child.RunE, "nil RunE should remain nil")
	})

	t.Run("skips subcommands with telemetry disabled", func(t *testing.T) {
		recorder := &telemetry.EventRecorderSpy{}

		root := &cobra.Command{Use: "gh"}
		child := &cobra.Command{
			Use:  "send-telemetry",
			RunE: func(cmd *cobra.Command, args []string) error { return nil },
		}
		cmdutil.DisableTelemetry(child)
		root.AddCommand(child)

		cmdutil.RecordTelemetryForSubcommands(root, nil, nil, recorder)
		require.NoError(t, child.RunE(child, nil))

		assert.Empty(t, recorder.Events, "disabled commands should not record telemetry")
	})
}
