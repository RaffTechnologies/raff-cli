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

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "backup",
		Aliases: []string{"backups"},
		Short:   "Manage VM backups",
	}
	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupGetCmd())
	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupRestoreCmd())
	cmd.AddCommand(newBackupDeleteCmd())
	cmd.AddCommand(newBackupScheduleCmd())
	return cmd
}

func newBackupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			items, _, err := c.Backups.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(items)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "SOURCE VM", "SIZE", "STATUS", "CREATED")
			for _, b := range items {
				vm := ""
				if b.ProductVM != nil {
					vm = b.ProductVM.String()
				}
				created := ""
				if b.CreatedAt != nil {
					created = b.CreatedAt.Format("2006-01-02 15:04")
				}
				t.AddRow(
					b.ID.String(),
					b.Name,
					vm,
					fmt.Sprintf("%d GB", b.StorageSize),
					b.Status,
					created,
				)
			}
			t.Flush()
			return nil
		},
	}
}

func newBackupGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <backup-id>",
		Short: "Get backup details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			b, _, err := c.Backups.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(b)
				output.PrintJSON(data)
				return nil
			}
			vm := ""
			if b.ProductVM != nil {
				vm = b.ProductVM.String()
			}
			expire := ""
			if b.ExpireDate != nil {
				expire = b.ExpireDate.Format("2006-01-02")
			}
			output.PrintDetail([][2]string{
				{"ID", b.ID.String()},
				{"Name", b.Name},
				{"Source VM", vm},
				{"Size", fmt.Sprintf("%d GB", b.StorageSize)},
				{"Status", b.Status},
				{"Expires", expire},
			})
			return nil
		},
	}
}

func newBackupCreateCmd() *cobra.Command {
	var vmID, name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Take an on-demand backup of a VM",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vid := openapi_types.UUID{}
			if err := vid.UnmarshalText([]byte(vmID)); err != nil {
				return fmt.Errorf("invalid --vm-id: %w", err)
			}
			req := &raff.CreateBackupRequest{VMID: vid}
			if name != "" {
				req.Name = &name
			}
			b, _, err := c.Backups.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(b)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Backup queued (id: %s, status: %s)\n", b.ID.String(), b.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&vmID, "vm-id", "", "Source VM UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Custom backup name (optional)")
	_ = cmd.MarkFlagRequired("vm-id")
	return cmd
}

func newBackupRestoreCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "restore <backup-id>",
		Short: "Restore a VM from a backup (overwrites current disk state)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Restore from backup %s? Current disk state will be lost. [y/N]: ", args[0])
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
			b, _, err := c.Backups.Restore(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(b)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Restore initiated (status: %s).\n", b.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newBackupDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <backup-id>",
		Short: "Delete a backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Delete backup %s? [y/N]: ", args[0])
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
			if _, err := c.Backups.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Backup deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

// === backup schedule subcommands ===

func newBackupScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule",
		Aliases: []string{"schedules"},
		Short:   "Manage recurring backup schedules",
	}
	cmd.AddCommand(newBackupScheduleListCmd())
	cmd.AddCommand(newBackupScheduleGetCmd())
	cmd.AddCommand(newBackupScheduleCreateCmd())
	cmd.AddCommand(newBackupScheduleUpdateCmd())
	cmd.AddCommand(newBackupScheduleDeleteCmd())
	return cmd
}

func newBackupScheduleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List backup schedules",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			items, _, err := c.BackupSchedules.List(context.Background(), nil)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(items)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "SOURCE VM", "FREQUENCY", "KEEP", "RUNTIME")
			for _, s := range items {
				vm := ""
				if s.ProductVM != nil {
					vm = s.ProductVM.String()
				}
				rt := ""
				if s.Runtime != nil {
					rt = *s.Runtime
				}
				t.AddRow(
					fmt.Sprintf("%d", s.ID),
					s.Name,
					vm,
					string(s.Type),
					fmt.Sprintf("%d", s.KeepCount),
					rt,
				)
			}
			t.Flush()
			return nil
		},
	}
}

func newBackupScheduleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <schedule-id>",
		Short: "Get backup schedule details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			s, _, err := c.BackupSchedules.Get(context.Background(), id)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			vm := ""
			if s.ProductVM != nil {
				vm = s.ProductVM.String()
			}
			rt := ""
			if s.Runtime != nil {
				rt = *s.Runtime
			}
			output.PrintDetail([][2]string{
				{"ID", fmt.Sprintf("%d", s.ID)},
				{"Name", s.Name},
				{"Source VM", vm},
				{"Frequency", string(s.Type)},
				{"Keep", fmt.Sprintf("%d", s.KeepCount)},
				{"Runtime", rt},
			})
			return nil
		},
	}
}

func newBackupScheduleCreateCmd() *cobra.Command {
	var vmID, freq, timeOfDay, day string
	var keep int
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a recurring backup schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vid := openapi_types.UUID{}
			if err := vid.UnmarshalText([]byte(vmID)); err != nil {
				return fmt.Errorf("invalid --vm-id: %w", err)
			}
			req := &raff.CreateBackupScheduleRequest{
				VMID:      vid,
				Type:      spec.CreateBackupScheduleRequestType(freq),
				KeepCount: keep,
				Time:      timeOfDay,
			}
			if day != "" {
				d := spec.CreateBackupScheduleRequestDayOfWeek(day)
				req.DayOfWeek = &d
			}
			s, _, err := c.BackupSchedules.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Backup schedule %q created (id: %d).\n", s.Name, s.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&vmID, "vm-id", "", "Source VM UUID (required)")
	cmd.Flags().StringVar(&freq, "frequency", "", "daily or weekly (required)")
	cmd.Flags().IntVar(&keep, "keep", 7, "Number of backups to retain before pruning")
	cmd.Flags().StringVar(&timeOfDay, "time", "08:00", "Time of day to run (e.g. 08:00, 8am)")
	cmd.Flags().StringVar(&day, "day", "", "Day of week for weekly schedules (e.g. Monday)")
	_ = cmd.MarkFlagRequired("vm-id")
	_ = cmd.MarkFlagRequired("frequency")
	return cmd
}

func newBackupScheduleUpdateCmd() *cobra.Command {
	var freq, timeOfDay, day string
	var keep int
	cmd := &cobra.Command{
		Use:   "update <schedule-id>",
		Short: "Update a backup schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.UpdateBackupScheduleRequest{}
			changed := false
			if cmd.Flags().Changed("frequency") {
				t := spec.UpdateBackupScheduleRequestType(freq)
				req.Type = &t
				changed = true
			}
			if cmd.Flags().Changed("keep") {
				req.KeepCount = &keep
				changed = true
			}
			if cmd.Flags().Changed("time") {
				req.Time = &timeOfDay
				changed = true
			}
			if cmd.Flags().Changed("day") {
				d := spec.UpdateBackupScheduleRequestDayOfWeek(day)
				req.DayOfWeek = &d
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --frequency, --keep, --time, --day required")
			}
			s, _, err := c.BackupSchedules.Update(context.Background(), id, req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Println("Backup schedule updated.")
			return nil
		},
	}
	cmd.Flags().StringVar(&freq, "frequency", "", "daily or weekly")
	cmd.Flags().IntVar(&keep, "keep", 0, "New retention count")
	cmd.Flags().StringVar(&timeOfDay, "time", "", "New time of day")
	cmd.Flags().StringVar(&day, "day", "", "New day of week")
	return cmd
}

func newBackupScheduleDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <schedule-id>",
		Short: "Delete a backup schedule (existing backups remain)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseIntArg(args[0], "schedule-id")
			if err != nil {
				return err
			}
			if !force {
				fmt.Printf("Delete schedule %d? [y/N]: ", id)
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
			if _, err := c.BackupSchedules.Delete(context.Background(), id); err != nil {
				return err
			}
			return printActionMessage("Backup schedule deleted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
