package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"MisakaMailClient/internal/output"
	"MisakaMailClient/internal/syspath"

	"github.com/spf13/cobra"
)

var installDir string

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Add the misaka-mail directory to the user PATH",
	Long: "Adds the directory containing the misaka-mail executable to the user's\n" +
		"PATH environment variable, so 'misaka-mail' can be run from any terminal.\n\n" +
		"On Windows this updates HKCU\\Environment\\Path and broadcasts the change so\n" +
		"new terminals pick it up immediately. On Unix it appends an export line to\n" +
		"~/.bashrc, ~/.zshrc, and ~/.profile (when present).",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := installDir
		if dir == "" {
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate executable: %w", err)
			}
			dir = filepath.Dir(exe)
		}
		dir = filepath.Clean(dir)

		added, err := syspath.AddToUserPath(dir)
		if err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"directory": dir,
				"added":     added,
			})
		}
		if added {
			fmt.Printf("Added %s to the user PATH.\nOpen a new terminal to run 'misaka-mail' from anywhere.\n", dir)
		} else {
			fmt.Printf("%s is already on the user PATH.\n", dir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&installDir, "dir", "", "directory to add (default: the running executable's directory)")
}
