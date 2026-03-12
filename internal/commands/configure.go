package commands

import (
	"bufio"
	"fmt"
	"os"

	"github.com/rafftechnologies/raff-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigureCmd() *cobra.Command {
	var profileName string

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure API credentials and defaults",
		Long:  "Set up a CLI profile with your API URL, API key, and optional defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if profileName == "" {
				profileName = "default"
			}

			existing := cfg.Profiles[profileName]
			reader := bufio.NewReader(os.Stdin)

			defaultURL := existing.APIURL
			if defaultURL == "" {
				defaultURL = "https://api.rafftechnologies.com"
			}

			apiURL := config.PromptInput(reader, "API URL", defaultURL)
			apiKey := config.PromptInput(reader, "API Key", existing.APIKey)
			projectID := config.PromptInput(reader, "Default Project ID (optional)", existing.ProjectID)

			cfg.Profiles[profileName] = config.Profile{
				APIURL:    apiURL,
				APIKey:    apiKey,
				ProjectID: projectID,
			}
			cfg.CurrentProfile = profileName

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Profile %q saved to %s\n", profileName, config.ConfigPath())
			return nil
		},
	}

	cmd.Flags().StringVar(&profileName, "profile", "", "Profile name (default: \"default\")")

	return cmd
}
