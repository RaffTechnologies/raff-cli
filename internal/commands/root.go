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

const Version = "0.1.0"

var (
	flagAPIURL    string
	flagAPIKey    string
	flagProjectID string
	flagOutput    string
)

var rootCmd = &cobra.Command{
	Use:     "raff",
	Short:   "Raff CLI — manage cloud resources from the terminal",
	Version: Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagProjectID, "project-id", "", "Default project ID (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table or json")

	rootCmd.AddCommand(newConfigureCmd())
	rootCmd.AddCommand(newProjectCmd())
	rootCmd.AddCommand(newVMCmd())
	rootCmd.AddCommand(newVPCCmd())
	rootCmd.AddCommand(newIPCmd())
	rootCmd.AddCommand(newSGCmd())
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
