package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rafftechnologies/raff-cli/internal/output"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/spf13/cobra"
)

func newAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps",
		Aliases: []string{"app"},
		Short:   "Manage Raff Apps services (PaaS)",
	}
	cmd.AddCommand(newAppsTiersCmd())
	cmd.AddCommand(newAppsListCmd())
	cmd.AddCommand(newAppsGetCmd())
	cmd.AddCommand(newAppsCreateCmd())
	cmd.AddCommand(newAppsDeleteCmd())
	cmd.AddCommand(newAppsDeployCmd())
	cmd.AddCommand(newAppsRollbackCmd())
	cmd.AddCommand(newAppsScaleCmd())
	cmd.AddCommand(newAppsPauseCmd())
	cmd.AddCommand(newAppsResumeCmd())
	cmd.AddCommand(newAppsLogsCmd())
	cmd.AddCommand(newAppsDeploymentsCmd())
	cmd.AddCommand(newAppsEnvCmd())
	cmd.AddCommand(newAppsDomainsCmd())
	cmd.AddCommand(newAppsUsageCmd())
	return cmd
}

func newAppsTiersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tiers",
		Short: "List Apps pricing tiers",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			tiers, _, err := c.AppServices.ListTiers(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(tiers)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "VCPU", "MEMORY (MiB)", "DISK (GiB)", "$/MO", "$/YR")
			for _, tr := range tiers {
				t.AddRow(
					strconv.Itoa(tr.ID),
					tr.Name,
					strconv.FormatFloat(tr.VCPU, 'g', -1, 64),
					strconv.Itoa(tr.MemoryMiB),
					strconv.Itoa(tr.EphemeralGiB),
					strconv.FormatFloat(tr.PricePerMonth, 'f', 2, 64),
					strconv.FormatFloat(tr.YearlyPrice, 'f', 2, 64),
				)
			}
			t.Flush()
			return nil
		},
	}
}

func newAppsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List app services",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			svcs, _, err := c.AppServices.List(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(svcs)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "TYPE", "TIER", "STATUS", "URL")
			for _, s := range svcs {
				t.AddRow(s.ServiceID, s.Slug, s.ServiceType, s.TierName, s.Status, s.URL)
			}
			t.Flush()
			return nil
		},
	}
}

func newAppsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <service>",
		Short: "Show one app service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			s, _, err := c.AppServices.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			output.PrintDetail([][2]string{
				{"ID", s.ServiceID},
				{"Name", s.Name},
				{"Slug", s.Slug},
				{"Type", s.ServiceType},
				{"Source", s.SourceType},
				{"Tier", s.TierName + " (#" + strconv.Itoa(s.TierID) + ")"},
				{"Replicas", strconv.Itoa(s.Replicas)},
				{"Status", s.Status},
				{"URL", s.URL},
				{"Region", s.Region},
				{"Billing", s.BillingType},
			})
			return nil
		},
	}
}

func newAppsCreateCmd() *cobra.Command {
	var name, description, serviceType, sourceType, builder, imageRef string
	var repo, branch, rootDir, dockerfile, startCommand, region, cronSchedule string
	var tierID, replicas int
	var deployNow bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an app service",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateAppServiceRequest{
				Name:           name,
				Description:    description,
				ServiceType:    serviceType,
				SourceType:     sourceType,
				Builder:        builder,
				ImageRef:       imageRef,
				RepoFullName:   repo,
				RepoBranch:     branch,
				RepoRootDir:    rootDir,
				DockerfilePath: dockerfile,
				StartCommand:   startCommand,
				TierID:         tierID,
				Replicas:       replicas,
				Region:         region,
				CronSchedule:   cronSchedule,
				DeployNow:      deployNow,
			}
			s, _, err := c.AppServices.Create(context.Background(), req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("App service %q created (id: %s, status: %s)\n", s.Name, s.ServiceID, s.Status)
			if s.URL != "" {
				fmt.Printf("URL: %s\n", s.URL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Service name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&serviceType, "type", "web", "Service type: web, private, worker, cron, or job")
	cmd.Flags().StringVar(&sourceType, "source", "", "Source type: git, image, template, or compose")
	cmd.Flags().StringVar(&builder, "builder", "", "Builder: buildpacks, dockerfile, or image")
	cmd.Flags().StringVar(&imageRef, "image", "", "Prebuilt image reference (for source=image)")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo (owner/name, for source=git)")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch")
	cmd.Flags().StringVar(&rootDir, "root-dir", "", "Source root directory")
	cmd.Flags().StringVar(&dockerfile, "dockerfile", "", "Dockerfile path (for builder=dockerfile)")
	cmd.Flags().StringVar(&startCommand, "start-command", "", "Override start command")
	cmd.Flags().IntVar(&tierID, "tier", 0, "Pricing tier ID (see `raff apps tiers`)")
	cmd.Flags().IntVar(&replicas, "replicas", 0, "Fixed replica count")
	cmd.Flags().StringVar(&region, "region", "", "Region")
	cmd.Flags().StringVar(&cronSchedule, "cron", "", "Cron schedule (for type=cron)")
	cmd.Flags().BoolVar(&deployNow, "deploy-now", false, "Queue a deployment immediately (image sources)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAppsDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <service>",
		Short: "Delete an app service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("Delete app service %s? [y/N]: ", args[0])
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
			if _, err := c.AppServices.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("App service deletion accepted.")
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newAppsDeployCmd() *cobra.Command {
	var channel, sourceBlobKey, imageRef string
	cmd := &cobra.Command{
		Use:   "deploy <service>",
		Short: "Trigger a new deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.CreateAppDeploymentRequest{
				Channel:       channel,
				SourceBlobKey: sourceBlobKey,
				ImageRef:      imageRef,
			}
			d, _, err := c.AppServices.CreateDeployment(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(d)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Deployment #%d queued (id: %s, status: %s)\n", d.DeploymentNumber, d.ID, d.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&channel, "channel", "cli", "Deploy channel")
	cmd.Flags().StringVar(&sourceBlobKey, "source-blob-key", "", "Uploaded source blob key")
	cmd.Flags().StringVar(&imageRef, "image", "", "Prebuilt image reference")
	return cmd
}

func newAppsRollbackCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "rollback <service>",
		Short: "Roll a service back to an earlier deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.AppServices.Rollback(context.Background(), args[0], &raff.RollbackAppServiceRequest{
				TargetDeploymentID: target,
			})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(d)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Rollback deployment #%d queued (id: %s)\n", d.DeploymentNumber, d.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "Target deployment ID to roll back to (required)")
	_ = cmd.MarkFlagRequired("target")
	return cmd
}

func newAppsScaleCmd() *cobra.Command {
	var replicas, minReplicas, maxReplicas, autoscalingTarget int
	var autoscalingMetric string
	var scaleToZero bool
	cmd := &cobra.Command{
		Use:   "scale <service>",
		Short: "Change replicas, autoscaling, or scale-to-zero",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			req := &raff.ScaleAppServiceRequest{
				Replicas:          replicas,
				ScaleToZero:       scaleToZero,
				AutoscalingMetric: autoscalingMetric,
				AutoscalingTarget: autoscalingTarget,
			}
			if cmd.Flags().Changed("min-replicas") || cmd.Flags().Changed("max-replicas") {
				req.AutoscalingSet = true
				req.Autoscaling = true
				req.MinReplicas = minReplicas
				req.MaxReplicas = maxReplicas
			}
			s, _, err := c.AppServices.Scale(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("Scaled %s (replicas: %d, autoscaling: %v)\n", s.Slug, s.Replicas, s.AutoscalingEnabled)
			return nil
		},
	}
	cmd.Flags().IntVar(&replicas, "replicas", 0, "Fixed replica count")
	cmd.Flags().IntVar(&minReplicas, "min-replicas", 0, "Autoscaling minimum replicas")
	cmd.Flags().IntVar(&maxReplicas, "max-replicas", 0, "Autoscaling maximum replicas")
	cmd.Flags().StringVar(&autoscalingMetric, "metric", "", "Autoscaling metric (e.g. cpu)")
	cmd.Flags().IntVar(&autoscalingTarget, "target", 0, "Autoscaling target value")
	cmd.Flags().BoolVar(&scaleToZero, "scale-to-zero", false, "Enable scale to zero")
	return cmd
}

func newAppsPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <service>",
		Short: "Pause an app service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, err := c.AppServices.Pause(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("App service paused.")
		},
	}
}

func newAppsResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <service>",
		Short: "Resume a paused app service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, err := c.AppServices.Resume(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("App service resumed.")
		},
	}
}

func newAppsLogsCmd() *cobra.Command {
	var level, search, since, until string
	var limit int
	cmd := &cobra.Command{
		Use:   "logs <service>",
		Short: "Show recent runtime logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			opts := &raff.AppLogOptions{
				Level:  level,
				Search: search,
				Since:  since,
				Until:  until,
				Limit:  limit,
			}
			lines, _, err := c.AppServices.ListLogs(context.Background(), args[0], opts)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(lines)
				output.PrintJSON(data)
				return nil
			}
			for _, l := range lines {
				fmt.Printf("%s  %s  %s\n", l.Timestamp, strings.ToUpper(l.Level), l.Message)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&level, "level", "", "Filter by level (info, warn, error)")
	cmd.Flags().StringVar(&search, "search", "", "Filter by substring")
	cmd.Flags().StringVar(&since, "since", "", "Start time (RFC3339)")
	cmd.Flags().StringVar(&until, "until", "", "End time (RFC3339)")
	cmd.Flags().IntVar(&limit, "limit", 200, "Maximum lines")
	return cmd
}

func newAppsDeploymentsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "deployments <service>",
		Short: "List a service's deployment history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			deps, _, err := c.AppServices.ListDeployments(context.Background(), args[0], limit)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(deps)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NUM", "CHANNEL", "STATUS", "COMMIT", "CREATED")
			for _, d := range deps {
				t.AddRow(d.ID, strconv.Itoa(d.DeploymentNumber), d.Channel, d.Status, d.CommitSHA, formatTime(d.CreatedAt))
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum results")
	return cmd
}

func newAppsEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage app service environment variables",
	}
	cmd.AddCommand(newAppsEnvListCmd())
	cmd.AddCommand(newAppsEnvSetCmd())
	cmd.AddCommand(newAppsEnvDeleteCmd())
	return cmd
}

func newAppsEnvListCmd() *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "list <service>",
		Short: "List environment variables (secrets masked)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			vars, _, err := c.AppServices.ListEnvVars(context.Background(), args[0], reveal)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(vars)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("KEY", "VALUE", "SECRET", "SYSTEM")
			for _, v := range vars {
				t.AddRow(v.Key, v.Value, strconv.FormatBool(v.IsSecret), strconv.FormatBool(v.IsSystem))
			}
			t.Flush()
			return nil
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal secret values (requires manage permission; audited)")
	return cmd
}

func newAppsEnvSetCmd() *cobra.Command {
	var value string
	var secret bool
	cmd := &cobra.Command{
		Use:   "set <service> <key>",
		Short: "Create or update an environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			_, _, err = c.AppServices.SetEnvVar(context.Background(), args[0], &raff.SetAppEnvVarRequest{
				Key:      args[1],
				Value:    value,
				IsSecret: secret,
			})
			if err != nil {
				return err
			}
			return printActionMessage("Environment variable set.")
		},
	}
	cmd.Flags().StringVar(&value, "value", "", "Variable value (required)")
	cmd.Flags().BoolVar(&secret, "secret", false, "Store as a secret (masked on read)")
	_ = cmd.MarkFlagRequired("value")
	return cmd
}

func newAppsEnvDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <service> <key>",
		Short: "Delete an environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, err := c.AppServices.DeleteEnvVar(context.Background(), args[0], args[1]); err != nil {
				return err
			}
			return printActionMessage("Environment variable deleted.")
		},
	}
}

func newAppsDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage custom domains for a web app service",
	}
	cmd.AddCommand(newAppsDomainsListCmd())
	cmd.AddCommand(newAppsDomainsAddCmd())
	cmd.AddCommand(newAppsDomainsDeleteCmd())
	return cmd
}

func newAppsDomainsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <service>",
		Short: "List custom domains",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			domains, _, err := c.AppServices.ListCustomDomains(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(domains)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "DOMAIN", "STATUS", "CNAME TARGET")
			for _, d := range domains {
				t.AddRow(d.ID, d.Domain, d.Status, d.CNAMETarget)
			}
			t.Flush()
			return nil
		},
	}
}

func newAppsDomainsAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <service> <domain>",
		Short: "Attach a custom domain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			d, _, err := c.AppServices.AddCustomDomain(context.Background(), args[0], &raff.AddAppCustomDomainRequest{
				Domain: args[1],
			})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(d)
				output.PrintJSON(data)
				return nil
			}
			output.PrintDetail([][2]string{
				{"ID", d.ID},
				{"Domain", d.Domain},
				{"Status", d.Status},
				{"CNAME target", d.CNAMETarget},
				{"Verify TXT name", d.VerificationTXTName},
				{"Verify TXT value", d.VerificationTXTValue},
			})
			return nil
		},
	}
}

func newAppsDomainsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <domain-id>",
		Short: "Remove a custom domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.AppServices.DeleteCustomDomain(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Custom domain deleted.")
		},
	}
}

func newAppsUsageCmd() *cobra.Command {
	var serviceID string
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show month-to-date Apps usage and estimated charge",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			u, _, err := c.AppServices.GetUsage(context.Background(), serviceID)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(u)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("SERVICE", "TIER", "REPLICA HOURS", "EST. COST ($)")
			for _, s := range u.Services {
				t.AddRow(
					s.ServiceName,
					strconv.Itoa(s.TierID),
					strconv.FormatFloat(s.ReplicaHours, 'f', 2, 64),
					strconv.FormatFloat(s.EstimatedCostUSD, 'f', 2, 64),
				)
			}
			t.Flush()
			fmt.Printf("\nTotal estimated cost: $%.2f (%s to %s)\n", u.TotalEstimatedCostUSD, u.PeriodStart, u.PeriodEnd)
			return nil
		},
	}
	cmd.Flags().StringVar(&serviceID, "service", "", "Scope to a single service ID")
	return cmd
}
