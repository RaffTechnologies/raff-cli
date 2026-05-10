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

func newProjectMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "member",
		Aliases: []string{"members"},
		Short:   "Manage project members",
		Long: `Manage members of a project.

The project is taken from --project-id (global flag), RAFF_PROJECT_ID,
or the active profile — same precedence as every other project-scoped
command (vm list, volume list, etc.).`,
	}
	cmd.AddCommand(newProjectMemberListCmd())
	cmd.AddCommand(newProjectMemberGetCmd())
	cmd.AddCommand(newProjectMemberAddCmd())
	cmd.AddCommand(newProjectMemberUpdateCmd())
	cmd.AddCommand(newProjectMemberRemoveCmd())
	return cmd
}

func newProjectMemberListCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List members of the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID()
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			var opts *raff.ProjectMemberListOptions
			if status != "" {
				s := spec.ListProjectMembersParamsStatus(status)
				opts = &raff.ProjectMemberListOptions{Status: &s}
			}
			members, _, err := c.ProjectMembers.List(context.Background(), pid, opts)
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
				role := ""
				if m.RoleName != nil {
					role = *m.RoleName
				}
				t.AddRow(m.ID.String(), string(m.Email), string(m.Status), role)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter: pending, active, suspended")
	return cmd
}

func newProjectMemberGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <member-id>",
		Short: "Get a project member's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID()
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			m, _, err := c.ProjectMembers.Get(context.Background(), pid, args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(m)
				output.PrintJSON(data)
				return nil
			}
			role := ""
			if m.RoleName != nil {
				role = *m.RoleName
			}
			output.PrintDetail([][2]string{
				{"ID", m.ID.String()},
				{"Email", string(m.Email)},
				{"Status", string(m.Status)},
				{"Role", role},
			})
			return nil
		},
	}
}

func newProjectMemberAddCmd() *cobra.Command {
	var roleID, targetUserID, apiKeyID string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an existing account user (or API key) to the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID()
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
			req := &raff.AddProjectMemberRequest{RoleID: rid}
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
			m, _, err := c.ProjectMembers.Add(context.Background(), pid, req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(m)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Project member added (id: %s)\n", m.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&roleID, "role-id", "", "Project-scoped role UUID (required)")
	cmd.Flags().StringVar(&targetUserID, "target-user-id", "", "Existing account user UUID (mutually exclusive with --api-key-id)")
	cmd.Flags().StringVar(&apiKeyID, "api-key-id", "", "API key UUID (mutually exclusive with --target-user-id)")
	_ = cmd.MarkFlagRequired("role-id")
	return cmd
}

func newProjectMemberUpdateCmd() *cobra.Command {
	var roleID, status string
	cmd := &cobra.Command{
		Use:   "update <member-id>",
		Short: "Update a project member's role or status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID()
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateProjectMemberRequest{}
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
			m, _, err := c.ProjectMembers.Update(context.Background(), pid, args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(m)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("Project member updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&roleID, "role-id", "", "New role UUID")
	cmd.Flags().StringVar(&status, "status", "", "New status: active or suspended")
	return cmd
}

func newProjectMemberRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <member-id>",
		Short: "Remove a member from the current project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := requireProjectID()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Are you sure you want to remove member %s from project %s? [y/N]: ", args[0], pid)
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
			if _, err := c.ProjectMembers.Remove(context.Background(), pid, args[0]); err != nil {
				return err
			}
			return printActionMessage("Project member removed.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
