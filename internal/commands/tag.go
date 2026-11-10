package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage account tags",
		Long: `Manage the account's tags. A tag is shared by every server that has it: a change
shows on all of them. Link a tag to a firewall with ` + "`raff security-group tags set`" + `
and the firewall goes on every server of that project with the tag.

Creating, updating and deleting tags needs the account.tags.manage permission
(account Owner and Admin by default).`,
	}
	cmd.AddCommand(newTagListCmd())
	cmd.AddCommand(newTagCreateCmd())
	cmd.AddCommand(newTagUpdateCmd())
	cmd.AddCommand(newTagDeleteCmd())
	return cmd
}

func newTagListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the account's tags (servers counted in RAFF_PROJECT_ID when set)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			tags, _, err := c.Tags.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(tags)
				output.PrintJSON(data)
				return nil
			}
			printAccountTags(tags)
			return nil
		},
	}
	return cmd
}

func newTagCreateCmd() *cobra.Command {
	var name, color string
	var priority int
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an account tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateTagRequest{Name: name}
			if cmd.Flags().Changed("color") {
				col := raff.TagColor(color)
				req.Color = &col
			}
			if cmd.Flags().Changed("priority") {
				req.Priority = raff.Int(priority)
			}
			tag, _, err := c.Tags.Create(context.Background(), req)
			if err != nil {
				return err
			}
			return printAccountTag(tag, "created")
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Tag name, 1-40 characters, unique in the account (required)")
	cmd.Flags().StringVar(&color, "color", "", "Colour: gray, red, orange, amber, green, blue, violet or pink (default gray)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Priority 1-9, 1 highest")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newTagUpdateCmd() *cobra.Command {
	var name, color string
	var priority int
	cmd := &cobra.Command{
		Use:   "update <tag-id>",
		Short: "Rename, recolour or reprioritise a tag on every server",
		Long: `Update a tag. The change shows on every server that has the tag.
--priority 0 removes the priority.

To find tag IDs, run:

  raff tag list`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateTagRequest{}
			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = raff.String(name)
				changed = true
			}
			if cmd.Flags().Changed("color") {
				col := raff.TagColor(color)
				req.Color = &col
				changed = true
			}
			if cmd.Flags().Changed("priority") {
				req.Priority = raff.Int(priority)
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --name, --color or --priority must be specified")
			}
			tag, _, err := c.Tags.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			return printAccountTag(tag, "updated")
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New tag name")
	cmd.Flags().StringVar(&color, "color", "", "New colour: gray, red, orange, amber, green, blue, violet or pink")
	cmd.Flags().IntVar(&priority, "priority", 0, "New priority 1-9 (0 removes it)")
	return cmd
}

func newTagDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <tag-id>",
		Short: "Delete a tag from the account and every server",
		Long: `Delete a tag. It is removed from every server and every firewall link.
Firewalls the tag put on servers stay on those servers.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Are you sure you want to delete tag %s from every server? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if _, err := c.Tags.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Tag deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func printAccountTags(tags []raff.Tag) {
	t := output.NewTable("ID", "NAME", "COLOR", "PRIORITY", "SERVERS", "FIREWALLS")
	for _, tag := range tags {
		t.AddRow(
			tag.ID.String(),
			tag.Name,
			string(tag.Color),
			tagPriority(tag.Priority),
			strconv.Itoa(tag.ServerCount),
			tagFirewallNames(tag.Firewalls),
		)
	}
	t.Flush()
}

func printAccountTag(tag *raff.Tag, verb string) error {
	if outputFormat() == output.FormatJSON {
		data, _ := json.Marshal(tag)
		output.PrintJSON(data)
		return nil
	}
	fmt.Printf("Tag %q %s (id: %s)\n", tag.Name, verb, tag.ID.String())
	return nil
}

// tagPriority is a tag's priority, or "-" when it has none.
func tagPriority(p *int) string {
	if p == nil || *p == 0 {
		return "-"
	}
	return strconv.Itoa(*p)
}

// tagFirewallNames lists the firewalls a tag carries, one per project, or "-" when none.
func tagFirewallNames(links []raff.TagFirewallLink) string {
	if len(links) == 0 {
		return "-"
	}
	names := make([]string, 0, len(links))
	for _, l := range links {
		names = append(names, l.SecurityGroupName)
	}
	return strings.Join(names, ", ")
}
