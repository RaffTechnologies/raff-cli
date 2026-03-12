package commands

import (
	"fmt"
	"os"

	"github.com/rafftechnologies/raff-cli/internal/client"
	"github.com/rafftechnologies/raff-cli/internal/config"
	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagAPIURL    string
	flagAPIKey    string
	flagAccountID string
	flagProjectID string
	flagOutput    string
)

var rootCmd = &cobra.Command{
	Use:     "raff",
	Short:   "Raff CLI — manage cloud resources from the terminal",
	Version: client.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "API base URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagAccountID, "account-id", "", "Account ID (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagProjectID, "project-id", "", "Project ID (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table or json")

	rootCmd.AddCommand(newConfigureCmd())
	rootCmd.AddCommand(newProjectCmd())
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return err
	}
	return nil
}

// newClient builds an API client from resolved config values.
func newClient() (*client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	profile := cfg.ActiveProfile()

	apiURL := config.Resolve(flagAPIURL, "RAFF_API_URL", profile.APIURL)
	apiKey := config.Resolve(flagAPIKey, "RAFF_API_KEY", profile.APIKey)
	accountID := config.Resolve(flagAccountID, "RAFF_ACCOUNT_ID", profile.AccountID)

	if apiURL == "" {
		apiURL = "https://api.rafftechnologies.com"
	}

	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured. Run 'raff configure' or set RAFF_API_KEY")
	}

	return client.New(apiURL, apiKey, accountID), nil
}

func outputFormat() output.Format {
	return output.ParseFormat(flagOutput)
}
