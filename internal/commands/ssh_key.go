package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSSHKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ssh-key",
		Aliases: []string{"ssh-keys"},
		Short:   "Manage SSH keys",
	}
	cmd.AddCommand(newSSHKeyListCmd())
	cmd.AddCommand(newSSHKeyGetCmd())
	cmd.AddCommand(newSSHKeyCreateCmd())
	cmd.AddCommand(newSSHKeyUpdateCmd())
	cmd.AddCommand(newSSHKeyDeleteCmd())
	return cmd
}

func newSSHKeyListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List SSH keys for the account",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			keys, _, err := c.SSHKeys.List(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(keys)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "TYPE", "FINGERPRINT")
			for _, k := range keys {
				fp := keyFingerprint(k.PublicKey)
				t.AddRow(k.ID.String(), k.Name, string(k.KeyType), fp)
			}
			t.Flush()
			return nil
		},
	}
}

func newSSHKeyGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key-id>",
		Short: "Get SSH key details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			k, _, err := c.SSHKeys.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			output.PrintDetail([][2]string{
				{"ID", k.ID.String()},
				{"Name", k.Name},
				{"Type", string(k.KeyType)},
				{"Public Key", k.PublicKey},
			})
			return nil
		},
	}
}

func newSSHKeyCreateCmd() *cobra.Command {
	var name, publicKey, publicKeyFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Register an SSH public key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if publicKeyFile != "" {
				data, err := os.ReadFile(publicKeyFile)
				if err != nil {
					return fmt.Errorf("read public key file: %w", err)
				}
				publicKey = strings.TrimSpace(string(data))
			}
			if publicKey == "" {
				return fmt.Errorf("either --public-key or --public-key-file is required")
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			k, _, err := c.SSHKeys.Create(context.Background(), &raff.CreateSSHKeyRequest{
				Name:      name,
				PublicKey: publicKey,
			})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("SSH key %q registered (id: %s)\n", k.Name, k.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "Full SSH public key string (e.g. \"ssh-ed25519 AAAA... user@host\")")
	cmd.Flags().StringVar(&publicKeyFile, "public-key-file", "", "Path to a file containing the public key (e.g. ~/.ssh/id_ed25519.pub)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSSHKeyUpdateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "update <key-id>",
		Short: "Rename an SSH key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			k, _, err := c.SSHKeys.Update(context.Background(), args[0], &raff.UpdateSSHKeyRequest{Name: name})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(k)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("SSH key renamed to %q\n", k.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name (required)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSSHKeyDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Delete an SSH key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Are you sure you want to delete SSH key %s? [y/N]: ", args[0])
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
			if _, err := c.SSHKeys.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("SSH key deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

// keyFingerprint returns the first 8 chars of the base64 portion of an
// SSH public key — quick visual identifier without bringing in crypto.
func keyFingerprint(publicKey string) string {
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return ""
	}
	if len(parts[1]) > 16 {
		return parts[1][:8] + "…" + parts[1][len(parts[1])-8:]
	}
	return parts[1]
}
