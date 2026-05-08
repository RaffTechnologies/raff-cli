package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"

	"github.com/rafftechnologies/raff-cli/internal/output"
	"github.com/spf13/cobra"
)

func newVMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Manage virtual machines",
	}

	cmd.AddCommand(newVMListCmd())
	cmd.AddCommand(newVMGetCmd())
	cmd.AddCommand(newVMCreateCmd())
	cmd.AddCommand(newVMDeleteCmd())
	cmd.AddCommand(newVMBulkDeleteCmd())
	cmd.AddCommand(newVMStartCmd())
	cmd.AddCommand(newVMStopCmd())
	cmd.AddCommand(newVMRebootCmd())
	cmd.AddCommand(newVMResizeCmd())
	cmd.AddCommand(newVMResizeDiskCmd())
	cmd.AddCommand(newVMRenameCmd())
	cmd.AddCommand(newVMResetPasswordCmd())
	cmd.AddCommand(newVMReinstallCmd())
	cmd.AddCommand(newVMFactoryResetCmd())
	cmd.AddCommand(newVMHardRebootCmd())
	cmd.AddCommand(newVMSaveImageCmd())
	cmd.AddCommand(newVMNetworksCmd())
	cmd.AddCommand(newVMVPCCmd())
	cmd.AddCommand(newVMIPCmd())
	cmd.AddCommand(newVMSGCmd())
	cmd.AddCommand(newVMTagsCmd())
	cmd.AddCommand(newVMNotesCmd())

	return cmd
}

func newVMListCmd() *cobra.Command {
	var projectID, region, status string
	var limit, offset int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all virtual machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			opts := &raff.VMListOptions{}
			if projectID != "" {
				id, err := uuid.Parse(projectID)
				if err != nil {
					return fmt.Errorf("invalid project-id: %w", err)
				}
				opts.ProjectID = &id
			}
			if region != "" {
				r := spec.ListVMsParamsRegion(region)
				opts.Region = &r
			}
			if status != "" {
				s := spec.ListVMsParamsStatus(status)
				opts.Status = &s
			}
			if limit > 0 {
				opts.Limit = raff.Int(limit)
			}
			if offset > 0 {
				opts.Offset = raff.Int(offset)
			}

			vms, _, err := c.VMs.List(context.Background(), opts)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vms)
				output.PrintJSON(data)
				return nil
			}

			t := output.NewTable("ID", "NAME", "STATUS", "CPU", "RAM", "STORAGE", "REGION", "IP", "CREATED")
			for _, vm := range vms {
				t.AddRow(
					vm.ID.String(),
					vm.Name,
					string(vm.Status),
					strconv.Itoa(vm.CPU),
					strconv.Itoa(vm.RAM),
					strconv.Itoa(vm.TotalStorage),
					string(vm.Region),
					raff.StringValue(vm.PublicIpv4Address),
					formatTime(vm.CreatedAt.Format("2006-01-02T15:04:05Z")),
				)
			}
			t.Flush()

			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Filter by project ID")
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of results to skip")

	return cmd
}

func newVMGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <vm-id>",
		Short: "Get virtual machine details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			vm, _, err := c.VMs.Get(context.Background(), args[0])
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vm)
				output.PrintJSON(data)
				return nil
			}

			printVMDetail(vm)
			return nil
		},
	}

	return cmd
}

func newVMCreateCmd() *cobra.Command {
	var name, templateID, region, password, extraStorageType, backupType, backupTime, backupDate, vpcID, vpcName, vpcCIDR string
	var pricingID, extraStorage int
	var sshKeys, tags []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new virtual machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			tmplID, err := uuid.Parse(templateID)
			if err != nil {
				return fmt.Errorf("invalid template-id: %w", err)
			}

			req := &raff.CreateVMRequest{
				Name:       name,
				TemplateID: tmplID,
				PricingID:  pricingID,
				Region:     spec.CreateVMRequestRegion(region),
			}

			if password != "" {
				req.Password = raff.String(password)
			}
			if len(sshKeys) > 0 {
				req.SSHKeys = &sshKeys
			}
			if extraStorage > 0 {
				req.ExtraStorage = raff.Int(extraStorage)
			}
			if extraStorageType != "" {
				st := spec.CreateVMRequestExtraStorageType(extraStorageType)
				req.ExtraStorageType = &st
			}
			if backupType != "" {
				bt := spec.CreateVMRequestBackupType(backupType)
				req.BackupType = &bt
			}
			if backupTime != "" {
				req.BackupTime = raff.String(backupTime)
			}
			if backupDate != "" {
				req.BackupDate = raff.String(backupDate)
			}
			if len(tags) > 0 {
				req.Tags = &tags
			}
			if vpcID != "" {
				id, err := uuid.Parse(vpcID)
				if err != nil {
					return fmt.Errorf("invalid vpc-id: %w", err)
				}
				req.VpcID = &id
			}
			if vpcName != "" {
				req.VpcName = raff.String(vpcName)
			}
			if vpcCIDR != "" {
				req.VpcCidr = raff.String(vpcCIDR)
			}

			vm, _, err := c.VMs.Create(context.Background(), req)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vm)
				output.PrintJSON(data)
				return nil
			}

			fmt.Println("VM created successfully.")
			printVMDetail(vm)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "VM name (required)")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Template ID (required)")
	cmd.Flags().IntVar(&pricingID, "pricing-id", 0, "Pricing plan ID (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region (required)")
	cmd.Flags().StringVar(&password, "password", "", "Root password")
	cmd.Flags().StringSliceVar(&sshKeys, "ssh-keys", nil, "SSH key IDs")
	cmd.Flags().IntVar(&extraStorage, "extra-storage", 0, "Extra storage in GB")
	cmd.Flags().StringVar(&extraStorageType, "extra-storage-type", "", "Extra storage type")
	cmd.Flags().StringVar(&backupType, "backup-type", "", "Backup type")
	cmd.Flags().StringVar(&backupTime, "backup-time", "", "Backup time")
	cmd.Flags().StringVar(&backupDate, "backup-date", "", "Backup date")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Tags")
	cmd.Flags().StringVar(&vpcID, "vpc-id", "", "VPC ID")
	cmd.Flags().StringVar(&vpcName, "vpc-name", "", "VPC name")
	cmd.Flags().StringVar(&vpcCIDR, "vpc-cidr", "", "VPC CIDR")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("template-id")
	_ = cmd.MarkFlagRequired("pricing-id")
	_ = cmd.MarkFlagRequired("region")

	return cmd
}

func newVMDeleteCmd() *cobra.Command {
	var force bool
	var volumeAction string
	var deleteVPC bool

	cmd := &cobra.Command{
		Use:   "delete <vm-id>",
		Short: "Delete a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmID := args[0]
			ctx := context.Background()

			c, err := newClient()
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("Are you sure you want to delete VM %s? [y/N]: ", vmID)
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			var req *raff.DeleteVMRequest
			if cmd.Flags().Changed("volume-action") || cmd.Flags().Changed("delete-vpc") {
				req = &raff.DeleteVMRequest{}
				if cmd.Flags().Changed("volume-action") {
					va := spec.DeleteVMRequestVolumeAction(volumeAction)
					req.VolumeAction = &va
				}
				if cmd.Flags().Changed("delete-vpc") {
					req.DeleteVpc = raff.Bool(deleteVPC)
				}
			}

			_, err = c.VMs.Delete(ctx, vmID, req)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				raw, _ := json.Marshal(map[string]any{"success": true, "message": "VM deleted successfully."})
				output.PrintJSON(raw)
				return nil
			}

			fmt.Println("VM deleted successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&volumeAction, "volume-action", "", "Volume action: detach or delete")
	cmd.Flags().BoolVar(&deleteVPC, "delete-vpc", false, "Delete associated VPC")

	return cmd
}

func newVMBulkDeleteCmd() *cobra.Command {
	var ids []string
	var force bool
	var volumeAction string
	var deleteVPC bool

	cmd := &cobra.Command{
		Use:   "bulk-delete",
		Short: "Delete multiple virtual machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			c, err := newClient()
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("Are you sure you want to delete %d VMs? [y/N]: ", len(ids))
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			parsedIDs := make([]uuid.UUID, 0, len(ids))
			for _, id := range ids {
				u, err := uuid.Parse(id)
				if err != nil {
					return fmt.Errorf("invalid VM ID %q: %w", id, err)
				}
				parsedIDs = append(parsedIDs, u)
			}

			req := &raff.BulkDeleteVMsRequest{Ids: parsedIDs}
			if cmd.Flags().Changed("volume-action") {
				va := spec.DeleteVMsBulkRequestVolumeAction(volumeAction)
				req.VolumeAction = &va
			}
			if cmd.Flags().Changed("delete-vpc") {
				req.DeleteVpc = raff.Bool(deleteVPC)
			}

			resp, _, err := c.VMs.BulkDelete(ctx, req)
			if err != nil {
				return err
			}

			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(resp)
				output.PrintJSON(data)
				return nil
			}

			fmt.Printf("Bulk delete: %d succeeded, %d failed (of %d total)\n",
				raff.IntValue(resp.SuccessCount),
				raff.IntValue(resp.FailedCount),
				raff.IntValue(resp.TotalCount))

			t := output.NewTable("ID", "SUCCESS", "MESSAGE")
			if resp.Results != nil {
				for _, r := range *resp.Results {
					id := ""
					if r.ID != nil {
						id = r.ID.String()
					}
					msg := raff.StringValue(r.Message)
					if errMsg := raff.StringValue(r.Error); errMsg != "" {
						msg = errMsg
					}
					t.AddRow(id, fmt.Sprintf("%v", raff.BoolValue(r.Success)), msg)
				}
			}
			t.Flush()

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&ids, "ids", nil, "VM IDs to delete (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&volumeAction, "volume-action", "", "Volume action: detach or delete")
	cmd.Flags().BoolVar(&deleteVPC, "delete-vpc", false, "Delete associated VPC")

	_ = cmd.MarkFlagRequired("ids")

	return cmd
}

func newVMStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <vm-id>",
		Short: "Start a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.Start(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VM started successfully.")
		},
	}
	return cmd
}

func newVMStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <vm-id>",
		Short: "Stop a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.Stop(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VM stopped successfully.")
		},
	}
	return cmd
}

func newVMRebootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reboot <vm-id>",
		Short: "Reboot a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.Reboot(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VM rebooted successfully.")
		},
	}
	return cmd
}

func newVMResizeCmd() *cobra.Command {
	var pricingID int

	cmd := &cobra.Command{
		Use:   "resize <vm-id>",
		Short: "Resize a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, err := c.VMs.Resize(context.Background(), args[0], &raff.ResizeVMRequest{PricingID: pricingID}); err != nil {
				return err
			}
			return printActionMessage("VM resized successfully.")
		},
	}

	cmd.Flags().IntVar(&pricingID, "pricing-id", 0, "New pricing plan ID (required)")
	_ = cmd.MarkFlagRequired("pricing-id")

	return cmd
}

func newVMResizeDiskCmd() *cobra.Command {
	var newSize int

	cmd := &cobra.Command{
		Use:   "resize-disk <vm-id>",
		Short: "Resize a virtual machine's disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, err := c.VMs.ResizeDisk(context.Background(), args[0], &raff.ResizeVMDiskRequest{NewSize: newSize}); err != nil {
				return err
			}
			return printActionMessage("VM disk resized successfully.")
		},
	}

	cmd.Flags().IntVar(&newSize, "new-size", 0, "New disk size in GB (required)")
	_ = cmd.MarkFlagRequired("new-size")

	return cmd
}

func newVMRenameCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "rename <vm-id>",
		Short: "Rename a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.Rename(context.Background(), args[0], &raff.RenameVMRequest{Name: name}); err != nil {
				return err
			}
			return printActionMessage("VM renamed successfully.")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New VM name (required)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newVMResetPasswordCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset-password <vm-id>",
		Short: "Reset the root password of a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("Are you sure you want to reset the password for VM %s? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if _, err := c.VMs.ResetPassword(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VM password reset successfully.")
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func newVMReinstallCmd() *cobra.Command {
	var templateID, password string
	var sshKeys []string
	var autoGeneratePassword, force bool

	cmd := &cobra.Command{
		Use:   "reinstall <vm-id>",
		Short: "Reinstall a virtual machine's operating system",
		Long: `Reinstall a virtual machine's operating system with a new template.

Authentication options (one of --password or --auto-generate-password is required):
  --password                 Provide a new root password directly
  --auto-generate-password   Auto-generate a password (emailed to account owner)
  --ssh-keys                 SSH key IDs for Linux VMs (can combine with either password option)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if password != "" && autoGeneratePassword {
				return fmt.Errorf("cannot use both --password and --auto-generate-password")
			}
			if password == "" && !autoGeneratePassword {
				return fmt.Errorf("must provide either --password or --auto-generate-password")
			}

			c, err := newClient()
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("Are you sure you want to reinstall VM %s? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			tmplID, err := uuid.Parse(templateID)
			if err != nil {
				return fmt.Errorf("invalid template-id: %w", err)
			}
			req := &raff.ReinstallVMRequest{TemplateID: tmplID}
			if password != "" {
				req.Password = raff.String(password)
			}
			if len(sshKeys) > 0 {
				req.SSHKeys = &sshKeys
			}
			if autoGeneratePassword {
				req.AutoGeneratePassword = raff.Bool(true)
			}

			if _, err := c.VMs.Reinstall(context.Background(), args[0], req); err != nil {
				return err
			}
			return printActionMessage("VM reinstalled successfully.")
		},
	}

	cmd.Flags().StringVar(&templateID, "template-id", "", "New template ID (required)")
	cmd.Flags().StringVar(&password, "password", "", "New root password (mutually exclusive with --auto-generate-password)")
	cmd.Flags().StringSliceVar(&sshKeys, "ssh-keys", nil, "SSH key IDs for Linux VMs (can combine with either password option)")
	cmd.Flags().BoolVar(&autoGeneratePassword, "auto-generate-password", false, "Auto-generate password and email it to the account owner")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	_ = cmd.MarkFlagRequired("template-id")

	return cmd
}

func newVMFactoryResetCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "factory-reset <vm-id>",
		Short: "Factory reset a virtual machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("Are you sure you want to factory reset VM %s? [y/N]: ", args[0])
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if _, err := c.VMs.FactoryReset(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VM factory reset successfully.")
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func printActionMessage(msg string) error {
	if outputFormat() == output.FormatJSON {
		raw, _ := json.Marshal(map[string]any{"success": true, "message": msg})
		output.PrintJSON(raw)
		return nil
	}
	fmt.Println(msg)
	return nil
}

func printVMDetail(vm *raff.VM) {
	pairs := [][2]string{
		{"ID", vm.ID.String()},
		{"Name", vm.Name},
		{"Status", string(vm.Status)},
		{"CPU", fmt.Sprintf("%d", vm.CPU)},
		{"RAM", fmt.Sprintf("%d GB", vm.RAM)},
		{"Storage", fmt.Sprintf("%d GB", vm.TotalStorage)},
		{"Template", vm.TemplateName + " " + vm.TemplateVersion},
		{"Region", string(vm.Region)},
		{"IPv4", raff.StringValue(vm.PublicIpv4Address)},
		{"Private IPv4", raff.StringValue(vm.PrivateIpv4Address)},
		{"Price/Hour", vm.PricePerHour},
		{"Pricing ID", fmt.Sprintf("%d", vm.PricingID)},
		{"Billing Type", billingTypeString(vm.BillingType)},
		{"Active", fmt.Sprintf("%v", vm.Active)},
		{"Created", vm.CreatedAt.Format("2006-01-02")},
	}
	output.PrintDetail(pairs)
}

func billingTypeString(b *spec.VMBillingType) string {
	if b == nil {
		return ""
	}
	return string(*b)
}

// ---- Hard reboot ----

func newVMHardRebootCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "hard-reboot <vm-id>",
		Short: "Force-reboot a VM (no graceful shutdown — data may be lost)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Hard reboot skips graceful shutdown. Unsaved data may be lost. Continue? [y/N]: ")
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if _, err := c.VMs.HardReboot(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("VM hard reboot initiated.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

// ---- Save image ----

func newVMSaveImageCmd() *cobra.Command {
	var name, description string
	var diskID, snapshotID int
	cmd := &cobra.Command{
		Use:   "save-image <vm-id>",
		Short: "Save a VM disk as a custom image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.SaveImageRequest{Name: name}
			if description != "" {
				req.Description = raff.String(description)
			}
			if cmd.Flags().Changed("disk-id") {
				req.DiskID = raff.Int(diskID)
			}
			if cmd.Flags().Changed("snapshot-id") {
				req.SnapshotID = raff.Int(snapshotID)
			}
			img, _, err := c.VMs.SaveImage(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(img)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Image %q created (id: %s)\n", img.Name, img.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Image name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().IntVar(&diskID, "disk-id", 0, "Disk to capture (0=OS, 1+=attached volume)")
	cmd.Flags().IntVar(&snapshotID, "snapshot-id", -1, "Snapshot ID (-1 = current live disk)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// ---- Networks ----

func newVMNetworksCmd() *cobra.Command {
	var netType string
	cmd := &cobra.Command{
		Use:   "networks <vm-id>",
		Short: "List a VM's network interfaces",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var opts *raff.ListVMNetworksOptions
			if netType != "" {
				if netType != "public" && netType != "vpc" && netType != "ipv6" {
					return fmt.Errorf("--type must be 'public', 'vpc', or 'ipv6'")
				}
				t := raff.VMNetworkType(netType)
				opts = &raff.ListVMNetworksOptions{Type: &t}
			}
			nets, _, err := c.VMs.ListNetworks(context.Background(), args[0], opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(nets)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("NIC", "TYPE", "NETWORK", "IP", "GATEWAY", "SECURITY GROUP")
			for _, n := range nets {
				t.AddRow(
					strconv.Itoa(n.NicID),
					string(n.Type),
					n.NetworkName,
					n.IP,
					raff.StringValue(n.Gateway),
					sgIDString(n.SecurityGroupID),
				)
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&netType, "type", "", "Filter by interface type: public, vpc, or ipv6")
	return cmd
}

func sgIDString(p *uuid.UUID) string {
	if p == nil {
		return ""
	}
	return p.String()
}

// ---- VPC attach/detach ----

func newVMVPCCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "vpc", Short: "Attach or detach a VM to/from a VPC"}
	cmd.AddCommand(newVMVPCAttachCmd())
	cmd.AddCommand(newVMVPCDetachCmd())
	return cmd
}

func newVMVPCAttachCmd() *cobra.Command {
	var vpcID, ip string
	cmd := &cobra.Command{
		Use:   "attach <vm-id>",
		Short: "Attach a VM to a VPC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vid, err := uuid.Parse(vpcID)
			if err != nil {
				return fmt.Errorf("invalid vpc-id: %w", err)
			}
			req := &raff.AttachVPCRequest{VpcID: vid}
			if ip != "" {
				req.IP = raff.String(ip)
			}
			if _, err := c.VMs.AttachVPC(context.Background(), args[0], req); err != nil {
				return err
			}
			return printActionMessage("VM attached to VPC.")
		},
	}
	cmd.Flags().StringVar(&vpcID, "vpc-id", "", "VPC ID (required)")
	cmd.Flags().StringVar(&ip, "ip", "", "Specific private IP (optional)")
	_ = cmd.MarkFlagRequired("vpc-id")
	return cmd
}

func newVMVPCDetachCmd() *cobra.Command {
	var nicID int
	cmd := &cobra.Command{
		Use:   "detach <vm-id>",
		Short: "Detach a VM's NIC from its VPC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.DetachVPC(context.Background(), args[0], nicID); err != nil {
				return err
			}
			return printActionMessage("VM detached from VPC.")
		},
	}
	cmd.Flags().IntVar(&nicID, "nic-id", -1, "NIC ID from `raff vm networks <vm-id>` (required)")
	_ = cmd.MarkFlagRequired("nic-id")
	return cmd
}

// ---- IP attach/detach ----

func newVMIPCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ip", Short: "Attach or detach a floating IP"}
	cmd.AddCommand(newVMIPAttachCmd())
	cmd.AddCommand(newVMIPDetachCmd())
	return cmd
}

func newVMIPAttachCmd() *cobra.Command {
	var ipID, ipType string
	var sgs []string
	cmd := &cobra.Command{
		Use:   "attach <vm-id>",
		Short: "Attach a floating IP to a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.AttachIPRequest{}
			if ipID != "" {
				id, err := uuid.Parse(ipID)
				if err != nil {
					return fmt.Errorf("invalid ip-id: %w", err)
				}
				req.IPID = &id
			}
			if ipType != "" {
				t := spec.AttachIPRequestType(ipType)
				req.Type = &t
			}
			if len(sgs) > 0 {
				parsed := make([]uuid.UUID, 0, len(sgs))
				for _, s := range sgs {
					u, err := uuid.Parse(s)
					if err != nil {
						return fmt.Errorf("invalid security-group: %w", err)
					}
					parsed = append(parsed, u)
				}
				req.SecurityGroups = &parsed
			}
			resp, _, err := c.VMs.AttachIP(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(resp)
				output.PrintJSON(data)
				return nil
			}
			ip := ""
			nic := ""
			if resp.Data != nil {
				if resp.Data.IPAddress != nil {
					ip = *resp.Data.IPAddress
				}
				if resp.Data.NicID != nil {
					nic = strconv.Itoa(*resp.Data.NicID)
				}
			}
			fmt.Printf("Attached %s (NIC %s)\n", ip, nic)
			return nil
		},
	}
	cmd.Flags().StringVar(&ipID, "ip-id", "", "Reserved IP ID (omit to auto-assign)")
	cmd.Flags().StringVar(&ipType, "type", "", "IP family for auto-assign: ipv4 or ipv6")
	cmd.Flags().StringSliceVar(&sgs, "security-groups", nil, "Security group IDs to apply to the new NIC")
	return cmd
}

func newVMIPDetachCmd() *cobra.Command {
	var nicID int
	cmd := &cobra.Command{
		Use:   "detach <vm-id>",
		Short: "Detach a floating IP from a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.DetachIP(context.Background(), args[0], nicID); err != nil {
				return err
			}
			return printActionMessage("Floating IP detached.")
		},
	}
	cmd.Flags().IntVar(&nicID, "nic-id", -1, "NIC ID from `raff vm networks <vm-id>` (required)")
	_ = cmd.MarkFlagRequired("nic-id")
	return cmd
}

// ---- Security group attach/detach ----

func newVMSGCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "sg", Short: "Attach or detach a security group"}
	cmd.AddCommand(newVMSGAttachCmd())
	cmd.AddCommand(newVMSGDetachCmd())
	return cmd
}

func newVMSGAttachCmd() *cobra.Command {
	var sgID string
	var nicID int
	cmd := &cobra.Command{
		Use:   "attach <vm-id>",
		Short: "Attach a security group to a VM NIC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			id, err := uuid.Parse(sgID)
			if err != nil {
				return fmt.Errorf("invalid security-group-id: %w", err)
			}
			req := &raff.AttachSecurityGroupRequest{SecurityGroupID: id, NicID: nicID}
			if _, err := c.VMs.AttachSecurityGroup(context.Background(), args[0], req); err != nil {
				return err
			}
			return printActionMessage("Security group attached.")
		},
	}
	cmd.Flags().StringVar(&sgID, "security-group-id", "", "Security group ID (required)")
	cmd.Flags().IntVar(&nicID, "nic-id", -1, "NIC ID (required)")
	_ = cmd.MarkFlagRequired("security-group-id")
	_ = cmd.MarkFlagRequired("nic-id")
	return cmd
}

func newVMSGDetachCmd() *cobra.Command {
	var sgID string
	var nicID int
	cmd := &cobra.Command{
		Use:   "detach <vm-id>",
		Short: "Detach a security group from a VM NIC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.VMs.DetachSecurityGroup(context.Background(), args[0], sgID, nicID); err != nil {
				return err
			}
			return printActionMessage("Security group detached.")
		},
	}
	cmd.Flags().StringVar(&sgID, "security-group-id", "", "Security group ID (required)")
	cmd.Flags().IntVar(&nicID, "nic-id", -1, "NIC ID (required)")
	_ = cmd.MarkFlagRequired("security-group-id")
	_ = cmd.MarkFlagRequired("nic-id")
	return cmd
}

// ---- Tags ----

func newVMTagsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tags", Short: "Manage VM tags"}
	cmd.AddCommand(newVMTagListCmd())
	cmd.AddCommand(newVMTagAddCmd())
	cmd.AddCommand(newVMTagUpdateCmd())
	cmd.AddCommand(newVMTagRemoveCmd())
	return cmd
}

func newVMTagListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <vm-id>",
		Short: "List tags on a VM (with their IDs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vm, _, err := c.VMs.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			var tags []raff.VMTag
			if vm.Tags != nil {
				tags = *vm.Tags
			}
			return printTags(tags)
		},
	}
	return cmd
}

func newVMTagAddCmd() *cobra.Command {
	var name string
	var priority int
	cmd := &cobra.Command{
		Use:   "add <vm-id>",
		Short: "Add a tag to a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.AddVMTagRequest{Name: name}
			if cmd.Flags().Changed("priority") {
				req.Priority = raff.Int(priority)
			}
			tags, _, err := c.VMs.AddTag(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			return printTags(tags)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Tag name (required)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Display priority")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newVMTagUpdateCmd() *cobra.Command {
	var name string
	var priority int
	var tagID string
	cmd := &cobra.Command{
		Use:   "update <vm-id>",
		Short: "Update a VM tag",
		Long: `Update a VM tag's name or priority.

To find tag IDs for a VM, run:

  raff vm tags list <vm-id>`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateVMTagRequest{}
			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = raff.String(name)
				changed = true
			}
			if cmd.Flags().Changed("priority") {
				req.Priority = raff.Int(priority)
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --name or --priority must be specified")
			}
			tags, _, err := c.VMs.UpdateTag(context.Background(), args[0], tagID, req)
			if err != nil {
				return err
			}
			return printTags(tags)
		},
	}
	cmd.Flags().StringVar(&tagID, "tag-id", "", "Tag ID to update (required)")
	cmd.Flags().StringVar(&name, "name", "", "New tag name")
	cmd.Flags().IntVar(&priority, "priority", 0, "New priority")
	_ = cmd.MarkFlagRequired("tag-id")
	return cmd
}

func newVMTagRemoveCmd() *cobra.Command {
	var tagID string
	cmd := &cobra.Command{
		Use:   "remove <vm-id>",
		Short: "Remove a tag from a VM",
		Long: `Remove a tag from a VM.

To find tag IDs for a VM, run:

  raff vm tags list <vm-id>`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			tags, _, err := c.VMs.RemoveTag(context.Background(), args[0], tagID)
			if err != nil {
				return err
			}
			return printTags(tags)
		},
	}
	cmd.Flags().StringVar(&tagID, "tag-id", "", "Tag ID to remove (required)")
	_ = cmd.MarkFlagRequired("tag-id")
	return cmd
}

func printTags(tags []raff.VMTag) error {
	if outputFormat() == output.FormatJSON {
		data, _ := json.Marshal(tags)
		output.PrintJSON(data)
		return nil
	}
	t := output.NewTable("ID", "NAME", "PRIORITY", "CREATED")
	for _, tag := range tags {
		t.AddRow(
			tag.ID,
			tag.Name,
			strconv.Itoa(tag.Priority),
			formatTime(tag.CreatedAt.Format("2006-01-02T15:04:05Z")),
		)
	}
	t.Flush()
	return nil
}

// ---- Notes ----

func newVMNotesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "notes", Short: "Manage VM notes"}
	cmd.AddCommand(newVMNotesGetCmd())
	cmd.AddCommand(newVMNotesSetCmd())
	cmd.AddCommand(newVMNotesUpdateCmd())
	cmd.AddCommand(newVMNotesAppendCmd())
	return cmd
}

func newVMNotesUpdateCmd() *cobra.Command {
	var noteType, content string
	cmd := &cobra.Command{
		Use:   "update <vm-id>",
		Short: "Replace an existing VM note (errors if no note exists)",
		Long: `Replace an existing personal or account note on a VM.

Returns 404 if no note has been written for this scope yet — use 'raff vm notes set'
to create one in that case.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			t := spec.UpsertVMNoteParamsType(noteType)
			note, _, err := c.VMs.UpdateNote(context.Background(), args[0], t, &raff.UpsertVMNoteRequest{Content: content})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(note)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Note updated (id: %s)\n", note.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&noteType, "type", "personal", "Note type: personal or account")
	cmd.Flags().StringVar(&content, "content", "", "New note content (required)")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newVMNotesAppendCmd() *cobra.Command {
	var noteType, content string
	cmd := &cobra.Command{
		Use:   "append <vm-id>",
		Short: "Append to a VM note without losing existing content",
		Long: `Append content to an existing personal or account note on a VM.

The new content is added to the end of the existing note, separated by a newline.
If no note exists yet, the new content becomes the entire note.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			t := spec.UpsertVMNoteParamsType(noteType)
			note, _, err := c.VMs.AppendNote(context.Background(), args[0], t, &raff.UpsertVMNoteRequest{Content: content})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(note)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Content appended (id: %s)\n", note.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&noteType, "type", "personal", "Note type: personal or account")
	cmd.Flags().StringVar(&content, "content", "", "Content to append (required)")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func newVMNotesGetCmd() *cobra.Command {
	var noteType string
	cmd := &cobra.Command{
		Use:   "get <vm-id>",
		Short: "Get personal and account notes for a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			var opts *raff.GetVMNotesOptions
			if noteType != "" {
				if noteType != "personal" && noteType != "account" {
					return fmt.Errorf("--type must be 'personal' or 'account'")
				}
				t := raff.VMNotesFilterType(noteType)
				opts = &raff.GetVMNotesOptions{Type: &t}
			}
			notes, _, err := c.VMs.GetNotes(context.Background(), args[0], opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(notes)
				output.PrintJSON(data)
				return nil
			}
			if notes.PersonalNote != nil {
				fmt.Println("--- Personal ---")
				fmt.Println(notes.PersonalNote.Content)
			}
			if notes.AccountNote != nil {
				fmt.Println("--- Account ---")
				fmt.Println(notes.AccountNote.Content)
			}
			if notes.PersonalNote == nil && notes.AccountNote == nil {
				fmt.Println("No notes.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&noteType, "type", "", "Filter to one scope: personal or account")
	return cmd
}

func newVMNotesSetCmd() *cobra.Command {
	var noteType, content string
	cmd := &cobra.Command{
		Use:   "set <vm-id>",
		Short: "Create or update a VM note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			t := spec.UpsertVMNoteParamsType(noteType)
			note, _, err := c.VMs.UpsertNote(context.Background(), args[0], t, &raff.UpsertVMNoteRequest{Content: content})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(note)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Note saved (id: %s)\n", note.ID.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&noteType, "type", "personal", "Note type: personal or account")
	cmd.Flags().StringVar(&content, "content", "", "Note content (required)")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}
