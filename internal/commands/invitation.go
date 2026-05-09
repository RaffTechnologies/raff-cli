package commands

import (
	"context"
	"encoding/json"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvitationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "invitation",
		Aliases: []string{"invitations", "invite"},
		Short:   "Send and cancel email invitations",
	}
	cmd.AddCommand(newInvitationCreateAccountCmd())
	cmd.AddCommand(newInvitationCreateProjectCmd())
	cmd.AddCommand(newInvitationCancelCmd())
	return cmd
}

func newInvitationCreateAccountCmd() *cobra.Command {
	var email, roleID string
	cmd := &cobra.Command{
		Use:   "create-account",
		Short: "Send an invitation to join the account",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			rid := openapi_types.UUID{}
			if err := rid.UnmarshalText([]byte(roleID)); err != nil {
				return fmt.Errorf("invalid --role-id: %w", err)
			}
			req := &raff.CreateInvitationRequest{
				Email:  openapi_types.Email(email),
				RoleID: rid,
			}
			inv, _, err := c.Invitations.CreateAccount(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(inv)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Account invitation sent to %s (id: %s)\n", email, inv.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email to invite (required)")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Account-scoped role UUID (required)")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("role-id")
	return cmd
}

func newInvitationCreateProjectCmd() *cobra.Command {
	var projectID, email, roleID string
	cmd := &cobra.Command{
		Use:   "create-project",
		Short: "Send an invitation to join a specific project",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			rid := openapi_types.UUID{}
			if err := rid.UnmarshalText([]byte(roleID)); err != nil {
				return fmt.Errorf("invalid --role-id: %w", err)
			}
			req := &raff.CreateInvitationRequest{
				Email:  openapi_types.Email(email),
				RoleID: rid,
			}
			inv, _, err := c.Invitations.CreateProject(context.Background(), projectID, req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(inv)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Project invitation sent to %s for project %s (id: %s)\n", email, projectID, inv.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project UUID (required)")
	cmd.Flags().StringVar(&email, "email", "", "Email to invite (required)")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Project-scoped role UUID (required)")
	_ = cmd.MarkFlagRequired("project-id")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("role-id")
	return cmd
}

func newInvitationCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <invitation-id>",
		Short: "Cancel a pending invitation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.Invitations.Cancel(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Invitation cancelled.")
		},
	}
}
