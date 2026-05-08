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

func newIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Manage floating IPs",
	}
	cmd.AddCommand(newIPListCmd())
	cmd.AddCommand(newIPGetCmd())
	cmd.AddCommand(newIPReserveCmd())
	cmd.AddCommand(newIPReleaseCmd())
	cmd.AddCommand(newIPChangeCmd())
	return cmd
}

func newIPListCmd() *cobra.Command {
	var reservedOnly bool
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List floating IPs",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.IPListOptions{}
			if cmd.Flags().Changed("reserved") {
				opts.Reserved = &reservedOnly
			}
			if status != "" {
				s := spec.ListIPsParamsStatus(status)
				opts.Status = &s
			}
			ips, _, err := c.IPs.List(context.Background(), opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(ips)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "ADDRESS", "TYPE", "STATUS", "RESERVED", "REGION")
			for _, ip := range ips {
				t.AddRow(
					ip.ID.String(),
					ip.IPAddress,
					string(ip.Type),
					string(ip.Status),
					fmt.Sprintf("%v", ip.Reserved),
					regionString(ip.Region),
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().BoolVar(&reservedOnly, "reserved", false, "Show only reserved IPs")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	return cmd
}

func newIPGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <ip-id>",
		Short: "Get floating IP details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			ip, _, err := c.IPs.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(ip)
				output.PrintJSON(data)
				return nil
			}
			printIPDetail(ip)
			return nil
		},
	}
	return cmd
}

func newIPReserveCmd() *cobra.Command {
	var ipType, region, billingPeriod string
	cmd := &cobra.Command{
		Use:   "reserve",
		Short: "Reserve a new floating IP",
		Long: `Reserve a new floating public IP held for your account until you release it.

Subscription accounts: the IP price is charged from account balance up front
for the chosen billing period (default: monthly). PAYG accounts: usage accrues
hourly with no upfront charge.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.ReserveIPRequest{}
			if ipType != "" {
				t := spec.ReserveIPRequestType(ipType)
				req.Type = &t
			}
			if region != "" {
				r := spec.ReserveIPRequestRegion(region)
				req.Region = &r
			}
			if billingPeriod != "" {
				bp := spec.ReserveIPRequestBillingPeriod(billingPeriod)
				req.BillingPeriod = &bp
			}
			ip, _, err := c.IPs.Reserve(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(ip)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("IP reserved successfully.")
			printIPDetail(ip)
			return nil
		},
	}
	cmd.Flags().StringVar(&ipType, "type", "", "IP family: IPv4 (default) or IPv6")
	cmd.Flags().StringVar(&region, "region", "", "Region (defaults to project's default)")
	cmd.Flags().StringVar(&billingPeriod, "billing-period", "", "Subscription period: monthly, yearly, twenty_four_month")
	return cmd
}

func newIPReleaseCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "release <ip-id>",
		Short: "Release a reserved IP",
		Long: `Release a reserved IP back to the pool.

Subscription IPs: remaining prepaid time is refunded pro-rata to your account
balance (hourly precision). PAYG IPs: hourly usage simply stops.

The IP must be detached from any VM first (raff vm ip detach <vm-id> --nic-id N).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Are you sure you want to release IP %s? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if _, err := c.IPs.Release(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("IP released successfully.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newIPChangeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change <ip-id>",
		Short: "Swap a reserved IP for a different one",
		Long: `Swap a reserved IP for a different one from the available pool. The subscription
stays the same — no extra charge, no refund. Useful if the current IP is
blacklisted somewhere.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			r, _, err := c.IPs.Change(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(r)
				output.PrintJSON(data)
				return nil
			}
			oldAddr := ""
			newAddr := ""
			if r.OldIP != nil {
				oldAddr = r.OldIP.IPAddress
			}
			if r.NewIP != nil {
				newAddr = r.NewIP.IPAddress
			}
			fmt.Printf("IP changed: %s → %s\n", oldAddr, newAddr)
			return nil
		},
	}
	return cmd
}

func printIPDetail(ip *raff.FloatingIP) {
	pairs := [][2]string{
		{"ID", ip.ID.String()},
		{"Address", ip.IPAddress},
		{"Type", string(ip.Type)},
		{"Status", string(ip.Status)},
		{"Reserved", fmt.Sprintf("%v", ip.Reserved)},
		{"Region", regionString(ip.Region)},
	}
	output.PrintDetail(pairs)
}

func regionString(r *spec.FloatingIPRegion) string {
	if r == nil {
		return ""
	}
	return string(*r)
}
