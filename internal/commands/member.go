package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "member",
		Aliases: []string{"members"},
		Short:   "Manage account-level members",
	}
	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberGetCmd())
	cmd.AddCommand(newMemberAddCmd())
	cmd.AddCommand(newMemberUpdateCmd())
	cmd.AddCommand(newMemberRemoveCmd())
	return cmd
}

func newMemberListCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List account members",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var opts *raff.MemberListOptions
			if status != "" {
				s := spec.ListMembersParamsStatus(status)
				opts = &raff.MemberListOptions{Status: &s}
			}
			members, _, err := c.Members.List(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(members)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "EMAIL", "STATUS", "ROLE")
			for _, m := range members {
				email := string(m.Email)
				role := ""
				if m.RoleName != nil {
					role = *m.RoleName
				}
				t.AddRow(m.ID.String(), email, string(m.Status), role)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter: pending, active, suspended")
	return cmd
}

func newMemberGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <member-id>",
		Short: "Get member details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			m, _, err := c.Members.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(m)
				output.PrintJSON(data)
				return nil
			}
			email := string(m.Email)
			role := ""
			if m.RoleName != nil {
				role = *m.RoleName
			}
			output.PrintDetail([][2]string{
				{"ID", m.ID.String()},
				{"Email", email},
				{"Status", string(m.Status)},
				{"Role", role},
			})
			return nil
		},
	}
}

func newMemberAddCmd() *cobra.Command {
	var email, roleID, targetUserID, apiKeyID string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a member to the account (use --email to invite, or --target-user-id / --api-key-id to add an existing user/key)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.AddMemberRequest{}
			if email != "" {
				e := openapi_types.Email(email)
				req.Email = &e
			}
			if targetUserID != "" {
				uid := openapi_types.UUID{}
				if err := uid.UnmarshalText([]byte(targetUserID)); err != nil {
					return fmt.Errorf("invalid --target-user-id: %w", err)
				}
				req.TargetUserID = &uid
			}
			if apiKeyID != "" {
				kid := openapi_types.UUID{}
				if err := kid.UnmarshalText([]byte(apiKeyID)); err != nil {
					return fmt.Errorf("invalid --api-key-id: %w", err)
				}
				req.APIKeyID = &kid
			}
			if roleID != "" {
				rid := openapi_types.UUID{}
				if err := rid.UnmarshalText([]byte(roleID)); err != nil {
					return fmt.Errorf("invalid --role-id: %w", err)
				}
				req.RoleID = &rid
			}
			m, _, err := c.Members.Add(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(m)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Member added (id: %s, status: %s)\n", m.ID.String(), m.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email to invite (mutually exclusive with --target-user-id, --api-key-id)")
	cmd.Flags().StringVar(&targetUserID, "target-user-id", "", "User UUID to add (mutually exclusive with --email)")
	cmd.Flags().StringVar(&apiKeyID, "api-key-id", "", "API key UUID to grant access (mutually exclusive with --email)")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Role UUID (required)")
	_ = cmd.MarkFlagRequired("role-id")
	return cmd
}

func newMemberUpdateCmd() *cobra.Command {
	var roleID, status string
	cmd := &cobra.Command{
		Use:   "update <member-id>",
		Short: "Update a member's role or status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateMemberRequest{}
			changed := false
			if cmd.Flags().Changed("role-id") {
				rid := openapi_types.UUID{}
				if err := rid.UnmarshalText([]byte(roleID)); err != nil {
					return fmt.Errorf("invalid --role-id: %w", err)
				}
				req.RoleID = &rid
				changed = true
			}
			if cmd.Flags().Changed("status") {
				s := spec.UpdateMemberRequestStatus(status)
				req.Status = &s
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --role-id, --status required")
			}
			m, _, err := c.Members.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(m)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("Member updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&roleID, "role-id", "", "New role UUID")
	cmd.Flags().StringVar(&status, "status", "", "New status: active or suspended")
	return cmd
}

func newMemberRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <member-id>",
		Short: "Remove a member from the account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Are you sure you want to remove member %s? [y/N]: ", args[0])
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
			if _, err := c.Members.Remove(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Member removed.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
