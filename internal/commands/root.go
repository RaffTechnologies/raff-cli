package commands

import (
	"fmt"
	"net/http"
	"os"
	"time"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/config"
	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

// Build-time injected via -ldflags by Makefile and goreleaser.
// Defaults are sane fallbacks for `go install` users (no ldflags).
var (
	Version = "0.2.0"
	Commit  = ""
	Date    = ""
)

// versionString renders "raff version X.Y.Z" or, when build metadata is
// available, "raff version X.Y.Z (commit abc1234, built 2026-05-09)".
func versionString() string {
	out := "raff version " + Version
	parts := []string{}
	if Commit != "" {
		short := Commit
		if len(short) > 7 {
			short = short[:7]
		}
		parts = append(parts, "commit "+short)
	}
	if Date != "" {
		parts = append(parts, "built "+Date)
	}
	if len(parts) > 0 {
		out += " (" + parts[0]
		for _, p := range parts[1:] {
			out += ", " + p
		}
		out += ")"
	}
	return out
}

var (
	flagAPIURL    string
	flagAPIKey    string
	flagProjectID string
	flagOutput    string
)

var rootCmd = &cobra.Command{
	Use:           "raff",
	Short:         "Raff CLI — manage cloud resources from the terminal",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(versionString())
	},
}

func init() {
	// Use the same rich format for `--version` and the `version` subcommand.
	rootCmd.SetVersionTemplate(versionString() + "\n")

	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagProjectID, "project-id", "", "Default project ID (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table or json")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newConfigureCmd())
	rootCmd.AddCommand(newProjectCmd())
	rootCmd.AddCommand(newVMCmd())
	rootCmd.AddCommand(newVPCCmd())
	rootCmd.AddCommand(newIPCmd())
	rootCmd.AddCommand(newSGCmd())
	rootCmd.AddCommand(newSSHKeyCmd())
	rootCmd.AddCommand(newAPIKeyCmd())
	rootCmd.AddCommand(newMemberCmd())
	rootCmd.AddCommand(newRoleCmd())
	rootCmd.AddCommand(newPermissionCmd())
	rootCmd.AddCommand(newInvitationCmd())
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return err
	}
	return nil
}

// newClient builds a raff-go API client from resolved config values.
func newClient() (*raff.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	profile := cfg.ActiveProfile()

	apiKey := config.Resolve(flagAPIKey, "RAFF_API_KEY", profile.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured. Run 'raff configure' or set RAFF_API_KEY")
	}

	opts := []raff.ClientOpt{
		raff.SetUserAgent("raff-cli/" + Version),
	}

	apiURL := config.Resolve(flagAPIURL, "RAFF_API_URL", profile.APIURL)
	if apiURL != "" {
		opts = append(opts, raff.SetBaseURL(apiURL))
	}

	projectID := config.Resolve(flagProjectID, "RAFF_PROJECT_ID", profile.ProjectID)
	if projectID != "" {
		opts = append(opts, raff.SetProjectID(projectID))
	}

	return raff.New(&http.Client{Timeout: 30 * time.Second}, apiKey, opts...), nil
}

func outputFormat() output.Format {
	return output.ParseFormat(flagOutput)
}
