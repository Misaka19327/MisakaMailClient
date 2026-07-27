package cmd

import (
	"MisakaMailClient/internal/message"

	"github.com/spf13/cobra"
)

// Footer flags (--footer / --no-footer) control the trailing notice that is
// appended to outgoing mail by default. They are shared by `send` and `reply`;
// only one of those runs per invocation, so a single set of package-level vars
// is safe.
var (
	footerText     string
	footerDisabled bool
)

// addFooterFlags registers the footer flags on a sending command. The default
// footer text is the built-in notice; --no-footer omits it and --footer sets a
// custom line (the wrapping style is fixed and not configurable).
func addFooterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&footerText, "footer", message.DefaultFooterText,
		"footer notice appended to the email body; use --no-footer to disable")
	cmd.Flags().BoolVar(&footerDisabled, "no-footer", false,
		"do not append the footer notice")
}

// resolveFooter returns the footer text to append to the outgoing message, or
// "" if the footer is disabled. --no-footer wins; otherwise the current
// --footer value (the default notice unless overridden) is used.
func resolveFooter() string {
	if footerDisabled {
		return ""
	}
	return footerText
}
