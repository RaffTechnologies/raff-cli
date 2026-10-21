package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/spf13/cobra"

	"github.com/rafftechnologies/raff-cli/internal/output"
)

// SSH keys on a running Linux server: read from the server and changed in it through its guest agent, without a
// restart. Every change is emailed to the account owner.

func newVMSSHKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ssh-keys",
		Aliases: []string{"ssh-key"},
		Short:   "List, add and remove the SSH keys on a running Linux server",
	}
	cmd.AddCommand(newVMSSHKeysListCmd())
	cmd.AddCommand(newVMSSHKeysAddCmd())
	cmd.AddCommand(newVMSSHKeysRemoveCmd())
	return cmd
}

func newVMSSHKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <vm-id>",
		Short: "List the keys in root's authorized_keys, read from the server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			keys, _, err := c.VMs.ListSSHKeys(context.Background(), args[0])
			if err != nil {
				return sshKeysError(err)
			}
			return printServerSSHKeys(keys)
		},
	}
}

func newVMSSHKeysAddCmd() *cobra.Command {
	var key, keyFile, saveAs string
	cmd := &cobra.Command{
		Use:   "add <vm-id>",
		Short: "Add an SSH key to root on a running Linux server",
		Long: `Add an SSH key to root's authorized_keys on a running Linux server, at once and without a restart.

--key is a saved key's ID (raff ssh-key list) or a public key; --key-file reads a public key from a .pub file.
With --save-as, a new public key is also saved to your account under that name.

  raff vm ssh-keys add <vm-id> --key 2f6b0c1e-...
  raff vm ssh-keys add <vm-id> --key-file ~/.ssh/id_ed25519.pub --save-as laptop`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if (key == "") == (keyFile == "") {
				return fmt.Errorf("give one of --key or --key-file")
			}
			if keyFile != "" {
				data, err := os.ReadFile(keyFile)
				if err != nil {
					return fmt.Errorf("reading %s: %w", keyFile, err)
				}
				key = strings.TrimSpace(string(data))
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.AddServerSSHKeyRequest{SSHKey: key}
			if saveAs != "" {
				req.SaveAs = raff.String(saveAs)
			}
			keys, _, err := c.VMs.AddSSHKey(context.Background(), args[0], req)
			if err != nil {
				return sshKeysError(err)
			}
			if outputFormat() != output.FormatJSON {
				if raff.IntValue(keys.Changed) > 0 {
					fmt.Println("Key added. The account owner gets an email.")
				} else {
					fmt.Println("The server already has this key.")
				}
			}
			return printServerSSHKeys(keys)
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "A saved key's ID, or a public key")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "A .pub file holding the public key")
	cmd.Flags().StringVar(&saveAs, "save-as", "", "Also save the key to your account under this name")
	return cmd
}

func newVMSSHKeysRemoveCmd() *cobra.Command {
	var fingerprint string
	var confirmLast, force bool
	cmd := &cobra.Command{
		Use:   "remove <vm-id>",
		Short: "Remove an SSH key from root on a running Linux server",
		Long: `Remove the key with this fingerprint from root's authorized_keys on a running Linux server, and from the
server's settings so it does not come back at the next restart.

Find the fingerprint with:

  raff vm ssh-keys list <vm-id>

Root's last key is removed only with --confirm-last: a server created with only an SSH key has password
sign-in turned off.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Remove key %s from VM %s? [y/N]: ", fingerprint, args[0])
				var answer string
				_, _ = fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			keys, _, err := c.VMs.RemoveSSHKey(context.Background(), args[0], fingerprint, &raff.RemoveServerSSHKeyOptions{ConfirmLast: confirmLast})
			if err != nil {
				return sshKeysError(err)
			}
			if outputFormat() != output.FormatJSON {
				fmt.Println("Key removed. The account owner gets an email.")
			}
			return printServerSSHKeys(keys)
		},
	}
	cmd.Flags().StringVar(&fingerprint, "fingerprint", "", "The key's SHA256 fingerprint (required)")
	cmd.Flags().BoolVar(&confirmLast, "confirm-last", false, "Remove the key even when it is root's last key")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	_ = cmd.MarkFlagRequired("fingerprint")
	return cmd
}

// sshKeysError adds what to do next to the refusals that have one.
func sshKeysError(err error) error {
	var apiErr *raff.ErrorResponse
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.Reason {
	case raff.SSHKeysLastKey:
		return fmt.Errorf("%s\nRun it again with --confirm-last to remove root's last key", apiErr.Message)
	case raff.SSHKeysSELinuxBlocked:
		return fmt.Errorf("%s", apiErr.Message)
	case raff.SSHKeysNoGuestAgent:
		return fmt.Errorf("%s\nInstall it in the server: apt install -y qemu-guest-agent (Debian, Ubuntu) or dnf install -y qemu-guest-agent (AlmaLinux, Rocky), then systemctl enable --now qemu-guest-agent", apiErr.Message)
	}
	return err
}

func printServerSSHKeys(keys *raff.ServerSSHKeys) error {
	if outputFormat() == output.FormatJSON {
		data, _ := json.Marshal(keys)
		output.PrintJSON(data)
		return nil
	}
	t := output.NewTable("FINGERPRINT", "TYPE", "COMMENT", "SAVED KEY", "OPTIONS")
	if keys.Keys != nil {
		for _, k := range *keys.Keys {
			options := ""
			if raff.BoolValue(k.HasOptions) {
				options = "yes"
			}
			t.AddRow(raff.StringValue(k.Fingerprint), raff.StringValue(k.Type), raff.StringValue(k.Comment), raff.StringValue(k.SavedKeyName), options)
		}
	}
	t.Flush()
	return nil
}
