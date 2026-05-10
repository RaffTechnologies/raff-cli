package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newSGCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security-group",
		Aliases: []string{"sg-resource"},
		Short: "Manage security groups",
	}
	cmd.AddCommand(newSGListCmd())
	cmd.AddCommand(newSGTemplatesCmd())
	cmd.AddCommand(newSGGetCmd())
	cmd.AddCommand(newSGCreateCmd())
	cmd.AddCommand(newSGUpdateCmd())
	cmd.AddCommand(newSGDeleteCmd())
	return cmd
}

func newSGListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List security groups (scoped to project via RAFF_PROJECT_ID)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			sgs, _, err := c.SecurityGroups.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(sgs)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "RULES", "ATTACHED VMS", "DESCRIPTION")
			for _, sg := range sgs {
				rules := 0
				if sg.Rules != nil {
					rules = len(sg.Rules)
				}
				t.AddRow(
					sg.ID.String(),
					sg.Name,
					fmt.Sprintf("%d", rules),
					fmt.Sprintf("%d", intValue(sg.VMCount)),
					raff.StringValue(sg.Description),
				)
			}
			t.Flush()
			return nil
		},
	}
	return cmd
}

func newSGTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "List pre-built security group templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			tmpls, _, err := c.SecurityGroups.Templates(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(tmpls)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "RULES", "DESCRIPTION")
			for _, tmpl := range tmpls {
				rules := 0
				if tmpl.Rules != nil {
					rules = len(tmpl.Rules)
				}
				t.AddRow(
					tmpl.ID,
					tmpl.Name,
					fmt.Sprintf("%d", rules),
					raff.StringValue(tmpl.Description),
				)
			}
			t.Flush()
			return nil
		},
	}
	return cmd
}

func newSGGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <sg-id>",
		Short: "Get security group details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			sg, _, err := c.SecurityGroups.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(sg)
				output.PrintJSON(data)
				return nil
			}
			pairs := [][2]string{
				{"ID", sg.ID.String()},
				{"Name", sg.Name},
				{"Description", raff.StringValue(sg.Description)},
				{"Attached VMs", fmt.Sprintf("%d", intValue(sg.VMCount))},
			}
			output.PrintDetail(pairs)
			fmt.Println()
			fmt.Println("Rules:")
			rt := output.NewTable("DIRECTION", "PROTOCOL", "RANGE", "IP")
			if sg.Rules != nil {
				for _, r := range sg.Rules {
					rt.AddRow(
						string(r.RuleType),
						string(r.Protocol),
						raff.StringValue(r.Range),
						raff.StringValue(r.IP),
					)
				}
			}
			rt.Flush()
			return nil
		},
	}
	return cmd
}

func newSGCreateCmd() *cobra.Command {
	var name, description, templateID string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a security group (use --template-id to clone a template's rules)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateSecurityGroupRequest{Name: name}
			if description != "" {
				req.Description = raff.String(description)
			}
			if templateID != "" {
				req.TemplateID = raff.String(templateID)
			}
			sg, _, err := c.SecurityGroups.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(sg)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Security group %q created (id: %s)\n", sg.Name, sg.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Security group name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Seed from a template (see `raff security-group templates`)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSGUpdateCmd() *cobra.Command {
	var name, description, rulesFile string
	cmd := &cobra.Command{
		Use:   "update <sg-id>",
		Short: "Update a security group's name, description, or rules",
		Long: `Update a security group. At least one of --name, --description, or --rules-file
must be provided. --rules-file replaces the entire rule set (the API does not
support partial rule updates).

Rules file format (JSON array, one object per rule):
  [
    {"rule_type":"inbound","protocol":"TCP","range":"22","ip":"","size":0},
    {"rule_type":"inbound","protocol":"TCP","range":"80,443","ip":"","size":0},
    {"rule_type":"outbound","protocol":"ALL"}
  ]`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateSecurityGroupRequest{}
			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = raff.String(name)
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description = raff.String(description)
				changed = true
			}
			if rulesFile != "" {
				data, err := os.ReadFile(rulesFile)
				if err != nil {
					return fmt.Errorf("read rules file: %w", err)
				}
				var rules []spec.SecurityGroupRule
				if err := json.Unmarshal(data, &rules); err != nil {
					return fmt.Errorf("parse rules file: %w", err)
				}
				req.Rules = &rules
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --name, --description, or --rules-file must be specified")
			}
			sg, _, err := c.SecurityGroups.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				out, _ := json.Marshal(sg)
				output.PrintJSON(out)
				return nil
			}
			fmt.Println("Security group updated successfully.")
			fmt.Printf("ID: %s\nName: %s\nRules: %d\n", sg.ID.String(), sg.Name, len(sg.Rules))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&rulesFile, "rules-file", "", "Path to JSON file containing the full rule set (replaces existing rules)")
	return cmd
}

func newSGDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <sg-id>",
		Short: "Delete a security group (must be detached from all VM NICs first)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Are you sure you want to delete security group %s? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if _, err := c.SecurityGroups.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Security group deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
