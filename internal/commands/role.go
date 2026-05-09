package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "role",
		Aliases: []string{"roles"},
		Short:   "Manage IAM roles",
	}
	cmd.AddCommand(newRoleListCmd())
	cmd.AddCommand(newRoleGetCmd())
	cmd.AddCommand(newRoleCreateCmd())
	cmd.AddCommand(newRoleUpdateCmd())
	cmd.AddCommand(newRoleDeleteCmd())
	return cmd
}

func newRoleListCmd() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List roles, optionally filtered by scope (account or project)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var opts *raff.RoleListOptions
			if scope != "" {
				s := spec.ListRolesParamsScope(scope)
				opts = &raff.RoleListOptions{Scope: &s}
			}
			roles, _, err := c.Roles.List(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(roles)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "SLUG", "SCOPE", "SYSTEM", "PERMISSIONS")
			for _, r := range roles {
				t.AddRow(
					r.ID.String(),
					r.Name,
					r.Slug,
					string(r.Scope),
					fmt.Sprintf("%v", r.IsSystem),
					fmt.Sprintf("%d", len(r.Permissions)),
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "Filter: account or project")
	return cmd
}

func newRoleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <role-id>",
		Short: "Get role details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			r, _, err := c.Roles.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(r)
				output.PrintJSON(data)
				return nil
			}
			desc := ""
			if r.Description != nil {
				desc = *r.Description
			}
			output.PrintDetail([][2]string{
				{"ID", r.ID.String()},
				{"Name", r.Name},
				{"Slug", r.Slug},
				{"Scope", string(r.Scope)},
				{"System", fmt.Sprintf("%v", r.IsSystem)},
				{"Description", desc},
				{"Permissions", strings.Join(r.Permissions, ", ")},
			})
			return nil
		},
	}
}

func newRoleCreateCmd() *cobra.Command {
	var name, slug, scope, description string
	var permissions []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom role",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateRoleRequest{
				Name:        name,
				Slug:        slug,
				Scope:       spec.CreateRoleRequestScope(scope),
				Permissions: permissions,
			}
			if description != "" {
				req.Description = &description
			}
			r, _, err := c.Roles.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(r)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Role %q created (id: %s)\n", r.Name, r.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe identifier (required)")
	cmd.Flags().StringVar(&scope, "scope", "", "Scope: account or project (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringSliceVar(&permissions, "permission", nil, "Permission name (repeatable, e.g. --permission vm.create --permission vm.read)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("slug")
	_ = cmd.MarkFlagRequired("scope")
	_ = cmd.MarkFlagRequired("permission")
	return cmd
}

func newRoleUpdateCmd() *cobra.Command {
	var name, description string
	var permissions []string
	cmd := &cobra.Command{
		Use:   "update <role-id>",
		Short: "Update a custom role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateRoleRequest{}
			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = &name
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
				changed = true
			}
			if cmd.Flags().Changed("permission") {
				req.Permissions = &permissions
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --name, --description, --permission required")
			}
			r, _, err := c.Roles.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(r)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("Role updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringSliceVar(&permissions, "permission", nil, "Replace permissions (repeatable)")
	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <role-id>",
		Short: "Delete a custom role (system roles cannot be deleted)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Are you sure you want to delete role %s? [y/N]: ", args[0])
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
			if _, err := c.Roles.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Role deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
