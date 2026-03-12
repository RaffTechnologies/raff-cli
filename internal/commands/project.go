package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())
	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectUpdateCmd())
	cmd.AddCommand(newProjectDeleteCmd())

	return cmd
}

func newProjectListCmd() *cobra.Command {
	var limit, offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			projects, _, err := c.Projects.List(context.Background(), &raff.ListOptions{
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(projects)
				output.PrintJSON(data)
				return nil
			}

			t := output.NewTable("ID", "NAME", "SLUG", "REGION", "DEFAULT", "ACTIVE", "CREATED")
			for _, p := range projects {
				t.AddRow(
					p.ID,
					p.Name,
					p.Slug,
					p.DefaultRegion,
					fmt.Sprintf("%v", p.IsDefault),
					fmt.Sprintf("%v", p.IsActive),
					formatTime(p.CreatedAt.Format("2006-01-02T15:04:05Z")),
				)
			}
			t.Flush()

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of results to skip")

	return cmd
}

func newProjectGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <project-id>",
		Short: "Get project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			project, _, err := c.Projects.Get(context.Background(), args[0])
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(project)
				output.PrintJSON(data)
				return nil
			}

			printProjectDetail(project)
			return nil
		},
	}

	return cmd
}

func newProjectCreateCmd() *cobra.Command {
	var name, description, region string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			project, _, err := c.Projects.Create(context.Background(), &raff.ProjectCreateRequest{
				Name:          name,
				Description:   description,
				DefaultRegion: region,
			})
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(project)
				output.PrintJSON(data)
				return nil
			}

			fmt.Println("Project created successfully.")
			printProjectDetail(project)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Project description")
	cmd.Flags().StringVar(&region, "region", "", "Default region (e.g. us-east)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newProjectUpdateCmd() *cobra.Command {
	var name, description, region string

	cmd := &cobra.Command{
		Use:   "update <project-id>",
		Short: "Update a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			req := &raff.ProjectUpdateRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = name
			}
			if cmd.Flags().Changed("description") {
				req.Description = description
			}
			if cmd.Flags().Changed("region") {
				req.DefaultRegion = region
			}

			if req.Name == "" && req.Description == "" && req.DefaultRegion == "" {
				return fmt.Errorf("at least one of --name, --description, or --region must be specified")
			}

			project, _, err := c.Projects.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(project)
				output.PrintJSON(data)
				return nil
			}

			fmt.Println("Project updated successfully.")
			printProjectDetail(project)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&description, "description", "", "Project description")
	cmd.Flags().StringVar(&region, "region", "", "Default region")

	return cmd
}

func newProjectDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			ctx := context.Background()

			c, err := newClient()
			if err != nil {
				return err
			}

			if !force {
				project, _, err := c.Projects.Get(ctx, projectID)
				if err != nil {
					return err
				}

				fmt.Printf("Are you sure you want to delete project %q? [y/N]: ", project.Name)
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			_, err = c.Projects.Delete(ctx, projectID)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				raw, _ := json.Marshal(map[string]any{
					"success": true,
					"message": "Project deleted successfully.",
				})
				output.PrintJSON(raw)
				return nil
			}

			fmt.Println("Project deleted successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func printProjectDetail(p *raff.Project) {
	pairs := [][2]string{
		{"ID", p.ID},
		{"Name", p.Name},
		{"Slug", p.Slug},
		{"Description", p.Description},
		{"Region", p.DefaultRegion},
		{"Default", fmt.Sprintf("%v", p.IsDefault)},
		{"Active", fmt.Sprintf("%v", p.IsActive)},
		{"Created", p.CreatedAt.Format("2006-01-02")},
	}
	output.PrintDetail(pairs)
}

func formatTime(t string) string {
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}
