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
	cmd.AddCommand(newAppsUpdateCmd())
	cmd.AddCommand(newAppsTemplatesCmd())
	cmd.AddCommand(newAppsGitHubCmd())
	cmd.AddCommand(newAppsLinksCmd())
	cmd.AddCommand(newAppsPreviewsCmd())
	cmd.AddCommand(newAppsEventsCmd())
	cmd.AddCommand(newAppsInstancesCmd())
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
	var healthCheck, cronTimezone, cronConcurrency, databaseID, githubConnection string
	var envPairs []string
	var tierID, replicas, port, minReplicas, maxReplicas int
	var deployNow, autoscale, scaleToZero bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an app service",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if sourceType == "" {
				switch {
				case imageRef != "":
					sourceType = "image"
				case repo != "":
					sourceType = "git"
				default:
					sourceType = "console" // source uploaded with `raff apps deploy --dir`
				}
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

				HTTPPort:             port,
				HealthCheckPath:      healthCheck,
				CronTimezone:         cronTimezone,
				CronConcurrency:      cronConcurrency,
				LinkDatabaseID:       databaseID,
				GitHubInstallationID: githubConnection,
				Autoscaling:          autoscale || cmd.Flags().Changed("min-replicas") || cmd.Flags().Changed("max-replicas"),
				MinReplicas:          minReplicas,
				MaxReplicas:          maxReplicas,
				ScaleToZero:          scaleToZero,
			}
			for _, pair := range envPairs {
				k, v, ok := strings.Cut(pair, "=")
				if !ok || k == "" {
					return fmt.Errorf("--env %q: use KEY=VALUE", pair)
				}
				req.EnvVars = append(req.EnvVars, raff.AppEnvVarInput{Key: k, Value: v})
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
	cmd.Flags().StringVar(&sourceType, "source", "", "Source: image (a Docker image or a template's image), git (a GitHub repository) or console (upload with `raff apps deploy --dir`); defaults from --image and --repo")
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
	cmd.Flags().BoolVar(&deployNow, "deploy-now", false, "Deploy right away (image and GitHub apps)")
	cmd.Flags().IntVar(&port, "port", 0, "Port the app listens on (web apps default to 8080; private apps need it)")
	cmd.Flags().StringVar(&healthCheck, "health-check", "", "HTTP path that answers 200 when the app is ready, such as /healthz")
	cmd.Flags().StringArrayVar(&envPairs, "env", nil, "Environment variable KEY=VALUE (repeatable)")
	cmd.Flags().StringVar(&databaseID, "database", "", "ID of a managed database to link (sets DATABASE_URL and the other DATABASE_* variables)")
	cmd.Flags().StringVar(&githubConnection, "github-connection", "", "GitHub connection ID for --source git (see `raff apps github`)")
	cmd.Flags().StringVar(&cronTimezone, "cron-timezone", "", "Cron time zone, such as Europe/London (default UTC)")
	cmd.Flags().StringVar(&cronConcurrency, "cron-concurrency", "", "When a run is due while the last one runs: forbid (default), allow or replace")
	cmd.Flags().BoolVar(&autoscale, "autoscale", false, "Autoscale between --min-replicas and --max-replicas")
	cmd.Flags().IntVar(&minReplicas, "min-replicas", 0, "Autoscaling minimum instances")
	cmd.Flags().IntVar(&maxReplicas, "max-replicas", 0, "Autoscaling maximum instances")
	cmd.Flags().BoolVar(&scaleToZero, "scale-to-zero", false, "Let the app sleep when idle (web and private apps)")
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
	var channel, sourceBlobKey, imageRef, dir string
	cmd := &cobra.Command{
		Use:   "deploy <service>",
		Short: "Deploy an app: upload a directory, switch to a new image, or redeploy what runs now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if dir != "" {
				archive, files, err := zipSourceDirWith(dir, appDefaultIgnores)
				if err != nil {
					return err
				}
				up, _, err := c.AppServices.RequestSourceUpload(context.Background(), args[0])
				if err != nil {
					return err
				}
				if err := c.AppServices.UploadSource(context.Background(), up.UploadURL, archive); err != nil {
					return err
				}
				if outputFormat() != output.FormatJSON {
					fmt.Printf("Uploaded %d files (%d KB)\n", files, len(archive)/1024)
				}
				sourceBlobKey = up.BlobKey
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
	cmd.Flags().StringVar(&dir, "dir", "", "Upload this directory and build it (files in .raffignore are skipped)")
	cmd.Flags().StringVar(&sourceBlobKey, "source-blob-key", "", "Uploaded source blob key")
	cmd.Flags().StringVar(&imageRef, "image", "", "Switch an image app to this image")
	cmd.MarkFlagsMutuallyExclusive("dir", "source-blob-key", "image")
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
	var scaleToZero, autoscale bool
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
			// With autoscaling_set the API takes autoscaling and scale to zero
			// exactly as sent, so a flag left out keeps the app's current value.
			minMax := cmd.Flags().Changed("min-replicas") || cmd.Flags().Changed("max-replicas")
			if minMax || cmd.Flags().Changed("autoscale") || cmd.Flags().Changed("scale-to-zero") ||
				cmd.Flags().Changed("metric") || cmd.Flags().Changed("target") {
				cur, _, err := c.AppServices.Get(context.Background(), args[0])
				if err != nil {
					return err
				}
				req.AutoscalingSet = true
				req.Autoscaling = cur.AutoscalingEnabled || minMax
				if cmd.Flags().Changed("autoscale") {
					req.Autoscaling = autoscale
				}
				req.ScaleToZero = cur.ScaleToZero
				if cmd.Flags().Changed("scale-to-zero") {
					req.ScaleToZero = scaleToZero
				}
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
			fmt.Printf("Scaled %s (replicas: %d, autoscaling: %v, scale to zero: %v)\n", s.Slug, s.Replicas, s.AutoscalingEnabled, s.ScaleToZero)
			return nil
		},
	}
	cmd.Flags().IntVar(&replicas, "replicas", 0, "Fixed replica count")
	cmd.Flags().IntVar(&minReplicas, "min-replicas", 0, "Autoscaling minimum replicas")
	cmd.Flags().IntVar(&maxReplicas, "max-replicas", 0, "Autoscaling maximum replicas")
	cmd.Flags().StringVar(&autoscalingMetric, "metric", "", "Autoscaling metric: concurrency (requests in progress per instance)")
	cmd.Flags().IntVar(&autoscalingTarget, "target", 0, "Autoscaling target value")
	cmd.Flags().BoolVar(&scaleToZero, "scale-to-zero", false, "Let the app sleep when idle (--scale-to-zero=false to keep it awake)")
	cmd.Flags().BoolVar(&autoscale, "autoscale", false, "Turn autoscaling on (--autoscale=false for a fixed count)")
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
			t := output.NewTable("KEY", "VALUE", "SECRET", "SYSTEM", "SCOPE")
			for _, v := range vars {
				scope := v.Scope
				if scope == "" {
					scope = raff.AppEnvScopeRuntime
				}
				t.AddRow(v.Key, v.Value, strconv.FormatBool(v.IsSecret), strconv.FormatBool(v.IsSystem), scope)
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
	var scope string
	cmd := &cobra.Command{
		Use:   "set <service> <key>",
		Short: "Create or update an environment variable",
		Long: `Create or update an environment variable.

By default a variable reaches the running container only. Frameworks that
inline configuration at build time need --scope build or --scope both:
Next.js NEXT_PUBLIC_*, Vite VITE_*, Create React App REACT_APP_*, SvelteKit
and Astro PUBLIC_*. Without it those values ship as empty strings.

Changing a build-scoped variable rebuilds the image rather than restarting
replicas, so it takes a full build to take effect. Build arguments are
recoverable from image history, so prefer runtime scope for credentials.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			switch scope {
			case raff.AppEnvScopeRuntime, raff.AppEnvScopeBuild, raff.AppEnvScopeBoth:
			default:
				return fmt.Errorf("invalid --scope %q: expected runtime, build or both", scope)
			}
			_, _, err = c.AppServices.SetEnvVar(context.Background(), args[0], &raff.SetAppEnvVarRequest{
				Key:      args[1],
				Value:    value,
				IsSecret: secret,
				Scope:    scope,
			})
			if err != nil {
				return err
			}
			if scope != raff.AppEnvScopeRuntime {
				return printActionMessage("Environment variable set. Rebuilding the image so the new value is baked in.")
			}
			return printActionMessage("Environment variable set.")
		},
	}
	cmd.Flags().StringVar(&value, "value", "", "Variable value (required)")
	cmd.Flags().BoolVar(&secret, "secret", false, "Store as a secret (masked on read)")
	cmd.Flags().StringVar(&scope, "scope", "runtime", "Where the value is injected: runtime, build or both")
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

func newAppsUpdateCmd() *cobra.Command {
	var name, description, startCommand, healthCheck, cronSchedule, cronTimezone, cronConcurrency, branch, rootDir, dockerfile string
	var tierID, port int
	var previews, redeploy bool
	cmd := &cobra.Command{
		Use:   "update <service>",
		Short: "Change an app's settings (a change to the running app rolls out as a new deploy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			f := cmd.Flags()
			req := &raff.UpdateAppServiceRequest{}
			for flag, dst := range map[string]**string{
				"name": &req.Name, "description": &req.Description, "start-command": &req.StartCommand,
				"health-check": &req.HealthCheckPath, "cron": &req.CronSchedule, "cron-timezone": &req.CronTimezone,
				"cron-concurrency": &req.CronConcurrency, "branch": &req.RepoBranch, "root-dir": &req.RepoRootDir,
				"dockerfile": &req.DockerfilePath,
			} {
				if f.Changed(flag) {
					v, _ := f.GetString(flag)
					*dst = raff.String(v)
				}
			}
			if f.Changed("tier") {
				req.TierID = raff.Int(tierID)
			}
			if f.Changed("port") {
				req.HTTPPort = raff.Int(port)
			}
			if f.Changed("previews") {
				req.PreviewsEnabled = raff.Bool(previews)
			}
			if f.Changed("redeploy") {
				req.Redeploy = raff.Bool(redeploy)
			}
			s, deploymentID, _, err := c.AppServices.Update(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(s)
				output.PrintJSON(data)
				return nil
			}
			fmt.Printf("App %s updated (status: %s)\n", s.Slug, s.Status)
			if deploymentID != "" {
				fmt.Printf("New deploy started (id: %s)\n", deploymentID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().IntVar(&tierID, "tier", 0, "Size (tier ID, see `raff apps tiers`); the app restarts on it")
	cmd.Flags().IntVar(&port, "port", 0, "Port the app listens on")
	cmd.Flags().StringVar(&startCommand, "start-command", "", "Start command")
	cmd.Flags().StringVar(&healthCheck, "health-check", "", "Health check path")
	cmd.Flags().StringVar(&cronSchedule, "cron", "", "Cron schedule (cron apps)")
	cmd.Flags().StringVar(&cronTimezone, "cron-timezone", "", "Cron time zone")
	cmd.Flags().StringVar(&cronConcurrency, "cron-concurrency", "", "forbid, allow or replace")
	cmd.Flags().StringVar(&branch, "branch", "", "GitHub branch to deploy")
	cmd.Flags().StringVar(&rootDir, "root-dir", "", "Source root directory")
	cmd.Flags().StringVar(&dockerfile, "dockerfile", "", "Dockerfile path")
	cmd.Flags().BoolVar(&previews, "previews", false, "Pull request previews on or off (web apps from GitHub)")
	cmd.Flags().BoolVar(&redeploy, "redeploy", false, "Redeploy even when the change does not need it (a change to how a live app runs always rolls out)")
	return cmd
}

func newAppsTemplatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "templates",
		Short: "List ready-to-run app templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			tpls, _, err := c.AppServices.ListTemplates(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(tpls)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "NAME", "IMAGE", "PORT", "MIN TIER", "POSTGRES")
			for _, tp := range tpls {
				pg := "no"
				if len(tp.Companions) > 0 {
					pg = "required"
					if tp.DatabaseOptional {
						pg = "optional"
					}
				}
				t.AddRow(tp.ID, tp.Name, tp.ImageRef, strconv.Itoa(tp.HTTPPort), strconv.Itoa(max(tp.MinTierID, 1)), pg)
			}
			t.Flush()
			return nil
		},
	}
}

func newAppsGitHubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "github",
		Short: "List GitHub connections (the ID goes in `raff apps create --github-connection`)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			conns, _, err := c.AppServices.ListGitHubConnections(context.Background())
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(conns)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "GITHUB", "TYPE", "STATUS")
			for _, g := range conns {
				t.AddRow(g.ID, g.GitHubLogin, g.AccountType, g.Status)
			}
			t.Flush()
			return nil
		},
	}
}

func newAppsLinksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "links",
		Short: "Link managed databases to an app",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list <service>",
		Short: "List the databases linked to an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			links, _, err := c.AppServices.ListBindings(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(links)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "DATABASE", "PREFIX", "STATUS")
			for _, l := range links {
				t.AddRow(l.ID, l.ResourceName, l.EnvPrefix, l.Status)
			}
			t.Flush()
			return nil
		},
	})
	var prefix string
	add := &cobra.Command{
		Use:   "add <service> <database-id>",
		Short: "Link a database (sets DATABASE_URL and the other DATABASE_* variables, then redeploys)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			l, _, err := c.AppServices.CreateBinding(context.Background(), args[0], &raff.CreateAppBindingRequest{
				BindingType: "database", ResourceID: args[1], EnvPrefix: prefix,
			})
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(l)
				output.PrintJSON(data)
				return nil
			}
			output.PrintDetail([][2]string{
				{"ID", l.ID},
				{"Database", l.ResourceName},
				{"Variables", strings.Join(l.InjectedKeys, ", ")},
				{"Status", l.Status},
			})
			return nil
		},
	}
	add.Flags().StringVar(&prefix, "prefix", "", "Variable prefix (default DATABASE)")
	cmd.AddCommand(add)
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <link-id>",
		Short: "Unlink a database (the app's database user and DATABASE_* variables are removed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.AppServices.DeleteBinding(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Database unlinked.")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "rotate <link-id>",
		Short: "Give the app new database credentials and redeploy it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, err := c.AppServices.RotateBinding(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Credentials rotated. The app redeploys with the new ones.")
		},
	})
	return cmd
}

func newAppsPreviewsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "previews",
		Short: "Pull request previews of a web app from GitHub",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list <service>",
		Short: "List an app's previews",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			previews, _, err := c.AppServices.ListPreviews(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(previews)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("ID", "PR", "BRANCH", "STATUS", "URL")
			for _, p := range previews {
				t.AddRow(p.ID, "#"+strconv.Itoa(p.PRNumber), p.Branch, p.Status, p.URL)
			}
			t.Flush()
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "redeploy <preview-id>",
		Short: "Build and deploy the pull request's latest commit again",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, _, _, err := c.AppServices.RedeployPreview(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Preview redeploy started.")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete <preview-id>",
		Short: "Delete a preview",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if _, err := c.AppServices.DeletePreview(context.Background(), args[0]); err != nil {
				return err
			}
			return printActionMessage("Preview deleted.")
		},
	})
	return cmd
}

func newAppsEventsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "events <service>",
		Short: "Show an app's activity timeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			events, _, err := c.AppServices.ListEvents(context.Background(), args[0], limit)
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(events)
				output.PrintJSON(data)
				return nil
			}
			for _, e := range events {
				fmt.Printf("%s  %s\n", e.CreatedAt, e.Message)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum events")
	return cmd
}

func newAppsInstancesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "instances <service>",
		Short: "List an app's running instances",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			instances, _, err := c.AppServices.ListInstances(context.Background(), args[0])
			if err != nil {
				return err
			}
			if outputFormat() == output.FormatJSON {
				data, _ := json.Marshal(instances)
				output.PrintJSON(data)
				return nil
			}
			t := output.NewTable("NAME", "PHASE", "READY", "RESTARTS", "STARTED")
			for _, in := range instances {
				t.AddRow(in.Name, in.Phase, strconv.FormatBool(in.Ready), strconv.Itoa(in.Restarts), in.StartedAt)
			}
			t.Flush()
			return nil
		},
	}
}
