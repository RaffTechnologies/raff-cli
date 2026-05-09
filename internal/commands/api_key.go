package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAPIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "api-key",
		Aliases: []string{"api-keys", "key", "keys"},
		Short:   "Manage API keys",
	}
	cmd.AddCommand(newAPIKeyListCmd())
	cmd.AddCommand(newAPIKeyGetCmd())
	cmd.AddCommand(newAPIKeyCreateCmd())
	cmd.AddCommand(newAPIKeyUpdateCmd())
	cmd.AddCommand(newAPIKeyRegenerateCmd())
	cmd.AddCommand(newAPIKeyRevokeCmd())
	return cmd
}

func newAPIKeyListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API keys for the account",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			keys, _, err := c.APIKeys.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(keys)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "PREFIX", "ACTIVE", "EXPIRES")
			for _, k := range keys {
				exp := "never"
				if k.ExpiresAt != nil {
					exp = k.ExpiresAt.Format("2006-01-02")
				}
				t.AddRow(k.ID.String(), k.Name, k.KeyPrefix, fmt.Sprintf("%v", k.IsActive), exp)
			}
			t.Flush()
			return nil
		},
	}
}

func newAPIKeyGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key-id>",
		Short: "Get API key details (secret is never returned — use create or regenerate)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			k, _, err := c.APIKeys.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			exp := "never"
			if k.ExpiresAt != nil {
				exp = k.ExpiresAt.Format(time.RFC3339)
			}
			output.PrintDetail([][2]string{
				{"ID", k.ID.String()},
				{"Name", k.Name},
				{"Prefix", k.KeyPrefix},
				{"Active", fmt.Sprintf("%v", k.IsActive)},
				{"Expires", exp},
			})
			return nil
		},
	}
}

func newAPIKeyCreateCmd() *cobra.Command {
	var name, rateLimitTier, expiresAt string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an API key (secret returned ONLY ONCE — copy it immediately)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateAPIKeyRequest{Name: name}
			if rateLimitTier != "" {
				t := spec.CreateAPIKeyRequestRateLimitTier(rateLimitTier)
				req.RateLimitTier = &t
			}
			if expiresAt != "" {
				t, err := time.Parse(time.RFC3339, expiresAt)
				if err != nil {
					return fmt.Errorf("--expires-at must be RFC3339 (e.g. 2026-12-31T23:59:59Z): %w", err)
				}
				req.ExpiresAt = &t
			}
			k, _, err := c.APIKeys.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("\nAPI key %q created.\n\n", k.Name)
			fmt.Println("⚠  Copy the secret NOW — it cannot be retrieved later:")
			fmt.Printf("    %s\n\n", k.Secret)
			fmt.Printf("ID:     %s\n", k.ID.String())
			fmt.Printf("Prefix: %s\n", k.KeyPrefix)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	cmd.Flags().StringVar(&rateLimitTier, "rate-limit-tier", "", "standard (default) or high")
	cmd.Flags().StringVar(&expiresAt, "expires-at", "", "Expiration in RFC3339 (e.g. 2026-12-31T23:59:59Z)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAPIKeyUpdateCmd() *cobra.Command {
	var name, rateLimitTier, expiresAt string
	var isActive bool
	cmd := &cobra.Command{
		Use:   "update <key-id>",
		Short: "Update an API key's metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateAPIKeyRequest{}
			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = &name
				changed = true
			}
			if cmd.Flags().Changed("rate-limit-tier") {
				t := spec.UpdateAPIKeyRequestRateLimitTier(rateLimitTier)
				req.RateLimitTier = &t
				changed = true
			}
			if cmd.Flags().Changed("active") {
				req.IsActive = &isActive
				changed = true
			}
			if cmd.Flags().Changed("expires-at") {
				t, err := time.Parse(time.RFC3339, expiresAt)
				if err != nil {
					return fmt.Errorf("--expires-at must be RFC3339: %w", err)
				}
				req.ExpiresAt = &t
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --name, --rate-limit-tier, --active, --expires-at required")
			}
			k, _, err := c.APIKeys.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("API key updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&rateLimitTier, "rate-limit-tier", "", "standard or high")
	cmd.Flags().BoolVar(&isActive, "active", true, "Activate (true) or deactivate (false) the key")
	cmd.Flags().StringVar(&expiresAt, "expires-at", "", "New expiration in RFC3339")
	return cmd
}

func newAPIKeyRegenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "regenerate <key-id>",
		Short: "Rotate an API key's secret (returns the new secret ONCE)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			k, _, err := c.APIKeys.Regenerate(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("\nAPI key %q regenerated.\n\n", k.Name)
			fmt.Println("⚠  Copy the new secret NOW — the old secret is now invalid:")
			fmt.Printf("    %s\n", k.Secret)
			return nil
		},
	}
}

func newAPIKeyRevokeCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "revoke <key-id>",
		Short: "Permanently revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Are you sure you want to revoke key %s? Any client using this key will fail. [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.APIKeys.Revoke(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("API key revoked.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
