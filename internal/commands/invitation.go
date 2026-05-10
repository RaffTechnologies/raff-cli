package commands

import (
	"context"
	"encoding/json"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvitationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "invitation",
		Aliases: []string{"invitations", "invite"},
		Short:   "Send and cancel email invitations",
	}
	cmd.AddCommand(newInvitationListCmd())
	cmd.AddCommand(newInvitationCreateAccountCmd())
	cmd.AddCommand(newInvitationCreateProjectCmd())
	cmd.AddCommand(newInvitationCancelCmd())
	return cmd
}

// newInvitationListCmd synthesizes a list of pending invitations from the
// Members / ProjectMembers endpoints. The public spec doesn't expose a
// dedicated invitation-list endpoint — pending invites surface as members
// with status="pending" and a populated invitation_id field. This command
// turns that into a flat list of {invitation_id, email, role, scope, ...}
// so users can discover IDs to pass to `raff invitation cancel`.
func newInvitationListCmd() *cobra.Command {
	var projectID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pending invitations (account-scoped by default; pass --project-id for project invites)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			type row struct {
				InvitationID string
				Email        string
				Role         string
				Scope        string
				ExpiresAt    string
			}
			var rows []row

			pending := spec.ListMembersParamsStatus("pending")

			if projectID != "" {
				pStatus := spec.ListProjectMembersParamsStatus("pending")
				members, _, err := c.ProjectMembers.List(
					context.Background(),
					projectID,
					&raff.ProjectMemberListOptions{Status: &pStatus},
				)
				if err != nil {
					return err
				}
				for _, m := range members {
					if m.InvitationID == nil {
						continue
					}
					role := ""
					if m.RoleName != nil {
						role = *m.RoleName
					}
					exp := ""
					if m.InvitationExpiresAt != nil {
						exp = m.InvitationExpiresAt.Format("2006-01-02")
					}
					rows = append(rows, row{
						InvitationID: m.InvitationID.String(),
						Email:        string(m.Email),
						Role:         role,
						Scope:        "project",
						ExpiresAt:    exp,
					})
				}
			} else {
				members, _, err := c.Members.List(
					context.Background(),
					&raff.MemberListOptions{Status: &pending},
				)
				if err != nil {
					return err
				}
				for _, m := range members {
					if m.InvitationID == nil {
						continue
					}
					role := ""
					if m.RoleName != nil {
						role = *m.RoleName
					}
					exp := ""
					if m.InvitationExpiresAt != nil {
						exp = m.InvitationExpiresAt.Format("2006-01-02")
					}
					rows = append(rows, row{
						InvitationID: m.InvitationID.String(),
						Email:        string(m.Email),
						Role:         role,
						Scope:        "account",
						ExpiresAt:    exp,
					})
				}
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(rows)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("INVITATION ID", "EMAIL", "ROLE", "SCOPE", "EXPIRES")
			for _, r := range rows {
				t.AddRow(r.InvitationID, r.Email, r.Role, r.Scope, r.ExpiresAt)
			}
			t.Flush()
			if len(rows) == 0 {
				fmt.Println("No pending invitations.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "List pending project invitations for this project (omit for account invites)")
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
	var email, roleID string
	cmd := &cobra.Command{
		Use:   "create-project",
		Short: "Send an invitation to join the current project (project from --project-id / RAFF_PROJECT_ID)",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID, err := requireProjectID()
			if err != nil {
				return err
			}
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
	cmd.Flags().StringVar(&email, "email", "", "Email to invite (required)")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Project-scoped role UUID (required)")
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
