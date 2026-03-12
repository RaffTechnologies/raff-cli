package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

type project struct {
	ID            string `json:"id"`
	AccountID     string `json:"account_id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Description   string `json:"description"`
	DefaultRegion string `json:"default_region"`
	IsDefault     bool   `json:"is_default"`
	IsActive      bool   `json:"is_active"`
	CreatedBy     string `json:"created_by"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

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

			path := fmt.Sprintf("/api/v1/projects?limit=%d&offset=%d", limit, offset)
			resp, err := c.Get(path)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				output.PrintJSON(resp.Data)
				return nil
			}

			var projects []project
			if err := json.Unmarshal(resp.Data, &projects); err != nil {
				return fmt.Errorf("parsing projects: %w", err)
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
					formatTime(p.CreatedAt),
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

			resp, err := c.Get("/api/v1/projects/" + args[0])
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				output.PrintJSON(resp.Data)
				return nil
			}

			var p project
			if err := json.Unmarshal(resp.Data, &p); err != nil {
				return fmt.Errorf("parsing project: %w", err)
			}

			printProjectDetail(p)
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

			body := map[string]string{"name": name}
			if description != "" {
				body["description"] = description
			}
			if region != "" {
				body["default_region"] = region
			}

			resp, err := c.Post("/api/v1/projects", body)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				output.PrintJSON(resp.Data)
				return nil
			}

			var p project
			if err := json.Unmarshal(resp.Data, &p); err != nil {
				return fmt.Errorf("parsing project: %w", err)
			}

			fmt.Println("Project created successfully.")
			printProjectDetail(p)
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

			body := make(map[string]string)
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("description") {
				body["description"] = description
			}
			if cmd.Flags().Changed("region") {
				body["default_region"] = region
			}

			if len(body) == 0 {
				return fmt.Errorf("at least one of --name, --description, or --region must be specified")
			}

			resp, err := c.Put("/api/v1/projects/"+args[0], body)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				output.PrintJSON(resp.Data)
				return nil
			}

			var p project
			if err := json.Unmarshal(resp.Data, &p); err != nil {
				return fmt.Errorf("parsing project: %w", err)
			}

			fmt.Println("Project updated successfully.")
			printProjectDetail(p)
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

			if !force {
				// Fetch project name for confirmation prompt
				c, err := newClient()
				if err != nil {
					return err
				}

				resp, err := c.Get("/api/v1/projects/" + projectID)
				if err != nil {
					return err
				}

				var p project
				if err := json.Unmarshal(resp.Data, &p); err != nil {
					return fmt.Errorf("parsing project: %w", err)
				}

				fmt.Printf("Are you sure you want to delete project %q? [y/N]: ", p.Name)
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

			resp, err := c.Delete("/api/v1/projects/" + projectID)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				raw, _ := json.Marshal(map[string]any{
					"success": resp.Success,
					"message": resp.Message,
				})
				output.PrintJSON(raw)
				return nil
			}

			msg := resp.Message
			if msg == "" {
				msg = "Project deleted successfully."
			}
			fmt.Println(msg)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func printProjectDetail(p project) {
	pairs := [][2]string{
		{"ID", p.ID},
		{"Name", p.Name},
		{"Slug", p.Slug},
		{"Description", p.Description},
		{"Region", p.DefaultRegion},
		{"Default", fmt.Sprintf("%v", p.IsDefault)},
		{"Active", fmt.Sprintf("%v", p.IsActive)},
		{"Created", formatTime(p.CreatedAt)},
	}
	output.PrintDetail(pairs)
}

func formatTime(t string) string {
	if len(t) >= 10 {
		return t[:10]
	}
	return t
}
