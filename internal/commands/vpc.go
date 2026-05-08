package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newVPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "Manage virtual private clouds",
	}

	cmd.AddCommand(newVPCListCmd())
	cmd.AddCommand(newVPCGetCmd())
	cmd.AddCommand(newVPCCreateCmd())
	cmd.AddCommand(newVPCUpdateCmd())
	cmd.AddCommand(newVPCDeleteCmd())
	cmd.AddCommand(newVPCCIDRSuggestionsCmd())

	return cmd
}

func newVPCListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPCs (scoped to current project via RAFF_PROJECT_ID / --project-id)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vpcs, _, err := c.VPCs.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vpcs)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "CIDR", "STATUS", "USED/TOTAL", "REGION", "CREATED")
			for _, v := range vpcs {
				usage := fmt.Sprintf("%d/%d", intValue(v.UsedIps), intValue(v.TotalIps))
				t.AddRow(
					v.ID.String(),
					v.Name,
					v.Cidr,
					v.Status,
					usage,
					string(v.Region),
					createdAtString(v.CreatedAt),
				)
			}
			t.Flush()
			return nil
		},
	}
	return cmd
}

func newVPCGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <vpc-id>",
		Short: "Get VPC details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vpc, _, err := c.VPCs.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vpc)
				output.PrintJSON(data)
				return nil
			}
			printVPCDetail(vpc)
			return nil
		},
	}
	return cmd
}

func newVPCCreateCmd() *cobra.Command {
	var name, cidr, region string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new VPC",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateVPCRequest{Name: name, Cidr: cidr}
			if region != "" {
				_ = region // Region passthrough — current spec only allows us-east; leave default.
			}
			vpc, _, err := c.VPCs.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vpc)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("VPC created successfully.")
			printVPCDetail(vpc)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "VPC name (required)")
	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR block, e.g. 10.0.0.0/24 (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region (defaults to us-east)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("cidr")
	return cmd
}

func newVPCUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "update <vpc-id>",
		Short: "Update a VPC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateVPCRequest{}
			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = raff.String(name)
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description = raff.String(description)
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --name or --description must be specified")
			}
			vpc, _, err := c.VPCs.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vpc)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("VPC updated successfully.")
			printVPCDetail(vpc)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New VPC name")
	cmd.Flags().StringVar(&description, "description", "", "New VPC description")
	return cmd
}

func newVPCDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <vpc-id>",
		Short: "Delete a VPC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Are you sure you want to delete VPC %s? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if _, err := c.VPCs.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VPC deleted successfully.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newVPCCIDRSuggestionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cidr-suggestions",
		Short: "Get suggested non-overlapping CIDR blocks for new VPCs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			r, _, err := c.VPCs.CIDRSuggestions(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(r)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Suggested: %s (%d IPs)\n", raff.StringValue(r.Suggested), intValue(r.SuggestedIps))
			if r.Alternatives != nil && len(*r.Alternatives) > 0 {
				fmt.Println("Alternatives:")
				t := output.NewTable("CIDR", "IPS", "SIZE")
				for _, alt := range *r.Alternatives {
					size := ""
					if alt.Size != nil {
						size = string(*alt.Size)
					}
					t.AddRow(raff.StringValue(alt.Cidr), strconv.Itoa(intValue(alt.AvailableIps)), size)
				}
				t.Flush()
			}
			return nil
		},
	}
	return cmd
}

func printVPCDetail(v *raff.VPC) {
	pairs := [][2]string{
		{"ID", v.ID.String()},
		{"Name", v.Name},
		{"CIDR", v.Cidr},
		{"Status", v.Status},
		{"Region", string(v.Region)},
		{"Used / Total IPs", fmt.Sprintf("%d / %d", intValue(v.UsedIps), intValue(v.TotalIps))},
		{"Gateway", raff.StringValue(v.Gateway)},
		{"DNS", raff.StringValue(v.DNS)},
		{"Created", createdAtString(v.CreatedAt)},
	}
	output.PrintDetail(pairs)
}

func intValue(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func createdAtString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
