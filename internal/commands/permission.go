package commands

import (
	"context"
	"encoding/json"

	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPermissionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permission",
		Aliases: []string{"permissions"},
		Short:   "List the permission catalog",
	}
	cmd.AddCommand(newPermissionListCmd())
	return cmd
}

func newPermissionListCmd() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all permissions, optionally filtered by scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var opts *spec.ListPermissionsParams
			if scope != "" {
				s := spec.ListPermissionsParamsScope(scope)
				opts = &spec.ListPermissionsParams{Scope: &s}
			}
			perms, _, err := c.Permissions.List(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(perms)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("NAME", "CATEGORY", "SCOPE", "DESCRIPTION")
			for _, p := range perms {
				desc := ""
				if p.Description != nil {
					desc = *p.Description
				}
				t.AddRow(p.Name, p.Category, string(p.Scope), desc)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "Filter by scope: account or project")
	return cmd
}
