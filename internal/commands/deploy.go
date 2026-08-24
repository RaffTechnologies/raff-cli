package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	raff "github.com/rafftechnologies/raff-go"

	"github.com/spf13/cobra"
)

const deployPollTimeout = 15 * time.Minute

func newDeployCmd() *cobra.Command {
	var flagName string
	var flagMessage string

	cmd := &cobra.Command{
		Use:   "deploy [dir]",
		Short: "Deploy a directory as a Raff Function",
		Long: `Deploy the current directory (or [dir]) as a serverless function.

Zero config: the runtime is detected from your files (requirements.txt,
package.json, go.mod, Dockerfile) and the function is created on first
deploy. Add raff.toml for settings (name, memory_mb, timeout_seconds,
min_scale, max_scale, [env]) and .raffignore to exclude files.
.git, node_modules, .venv, __pycache__ and .env are always excluded.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			if info, err := os.Stat(absDir); err != nil || !info.IsDir() {
				return fmt.Errorf("not a directory: %s", absDir)
			}

			tomlCfg, err := loadRaffToml(absDir)
			if err != nil {
				return err
			}

			name := flagName
			if name == "" && tomlCfg != nil {
				name = tomlCfg.Name
			}
			if name == "" {
				name = slugifyName(filepath.Base(absDir))
			}
			if name == "" {
				return fmt.Errorf("could not derive a function name — pass --name or set name in raff.toml")
			}

			// Uploads can outlive the default 30s client timeout.
			c, err := newClientWithTimeout(5 * time.Minute)
			if err != nil {
				return err
			}
			ctx := context.Background()

			fn, err := resolveOrCreateFunction(ctx, c, absDir, name, tomlCfg)
			if err != nil {
				return err
			}

			if err := applyTomlEnv(ctx, c, fn.Slug, tomlCfg); err != nil {
				return err
			}

			fmt.Printf("Packing %s…\n", absDir)
			zipData, fileCount, err := zipSourceDir(absDir)
			if err != nil {
				return err
			}
			fmt.Printf("→ %d files, %.1f KB\n", fileCount, float64(len(zipData))/1024)

			upload, _, err := c.Functions.RequestSourceUpload(ctx, fn.Slug)
			if err != nil {
				return err
			}
			fmt.Println("Uploading source…")
			if err := c.Functions.UploadSource(ctx, upload.UploadURL, zipData); err != nil {
				return err
			}

			message := flagMessage
			if message == "" {
				message = "raff deploy"
			}
			dep, _, err := c.Functions.CreateDeployment(ctx, fn.Slug, &raff.CreateFunctionDeploymentRequest{
				Channel:       "cli",
				SourceBlobKey: upload.SourceBlobKey,
				Message:       message,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Deployment %s started\n\n", dep.ID)

			return followDeployment(ctx, c, fn.Slug, dep.ID)
		},
	}

	cmd.Flags().StringVar(&flagName, "name", "", "Function name (default: raff.toml name or directory name)")
	cmd.Flags().StringVarP(&flagMessage, "message", "m", "", "Deployment message")
	return cmd
}

// resolveOrCreateFunction finds the function by name/slug or creates it on
// first deploy (runtime detected from files unless raff.toml pins it).
func resolveOrCreateFunction(ctx context.Context, c *raff.Client, dir, name string, tomlCfg *raffToml) (*raff.Function, error) {
	fn, _, err := c.Functions.Get(ctx, name)
	if err == nil {
		fmt.Printf("Deploying to %s (%s, rev-%d)\n", fn.Slug, fn.Runtime, fn.CurrentRevisionNumber)
		noteSettingsDrift(fn, tomlCfg)
		return fn, nil
	}
	if apiErr, ok := err.(*raff.ErrorResponse); !ok || apiErr.StatusCode != 404 {
		return nil, err
	}

	runtime := ""
	if tomlCfg != nil {
		runtime = tomlCfg.Runtime
	}
	if runtime == "" {
		runtime, err = detectRuntime(dir)
		if err != nil {
			return nil, err
		}
	}
	req := &raff.CreateFunctionRequest{
		Name:       name,
		Runtime:    runtime,
		SourceType: "cli",
	}
	if tomlCfg != nil {
		req.MemoryMB = tomlCfg.MemoryMB
		req.TimeoutSeconds = tomlCfg.TimeoutSeconds
		req.ExtendedTimeout = tomlCfg.ExtendedTimeout
		req.MinScale = tomlCfg.MinScale
		req.MaxScale = tomlCfg.MaxScale
	}
	fmt.Printf("Creating function %s (%s)…\n", name, runtime)
	fn, _, err = c.Functions.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	if tomlCfg != nil && (len(tomlCfg.Triggers) > 0 || len(tomlCfg.Bindings) > 0) {
		fmt.Println("note: triggers/bindings in raff.toml are managed in the dashboard for now")
	}
	return fn, nil
}

// noteSettingsDrift warns when raff.toml settings differ from the deployed
// function — there is no settings-update API on the CLI path yet.
func noteSettingsDrift(fn *raff.Function, tomlCfg *raffToml) {
	if tomlCfg == nil {
		return
	}
	drift := func(field string, want, have int) {
		if want != 0 && want != have {
			fmt.Printf("note: raff.toml %s=%d differs from deployed %d — change it in the dashboard\n", field, want, have)
		}
	}
	drift("memory_mb", tomlCfg.MemoryMB, fn.MemoryMB)
	drift("timeout_seconds", tomlCfg.TimeoutSeconds, fn.TimeoutSeconds)
	drift("max_scale", tomlCfg.MaxScale, fn.MaxScale)
	if tomlCfg.MinScale != 0 && tomlCfg.MinScale != fn.MinScale {
		fmt.Printf("note: raff.toml min_scale=%d differs from deployed %d — change it in the dashboard\n", tomlCfg.MinScale, fn.MinScale)
	}
}

// applyTomlEnv sets [env] vars from raff.toml that differ from the server.
// Plain vars only — secrets belong in the dashboard, never in the repo.
func applyTomlEnv(ctx context.Context, c *raff.Client, ref string, tomlCfg *raffToml) error {
	if tomlCfg == nil || len(tomlCfg.Env) == 0 {
		return nil
	}
	existing, _, err := c.Functions.ListEnvVars(ctx, ref, true)
	if err != nil {
		return fmt.Errorf("reading current env vars: %w", err)
	}
	current := map[string]string{}
	for _, v := range existing {
		current[v.Key] = v.Value
	}
	for key, value := range tomlCfg.Env {
		if current[key] == value {
			continue
		}
		fmt.Printf("Setting env %s\n", key)
		if _, err := c.Functions.SetEnvVar(ctx, ref, &raff.SetFunctionEnvVarRequest{Key: key, Value: value}); err != nil {
			return fmt.Errorf("setting env %s: %w", key, err)
		}
	}
	return nil
}

// followDeployment polls status + streams new build-log lines until the
// deployment is ready or failed.
func followDeployment(ctx context.Context, c *raff.Client, ref, deploymentID string) error {
	deadline := time.Now().Add(deployPollTimeout)
	printed := 0
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for deployment %s — check the dashboard", deploymentID)
		}

		lines, _, _, err := c.Functions.GetDeploymentLogs(ctx, ref, deploymentID)
		if err == nil {
			for ; printed < len(lines); printed++ {
				fmt.Printf("  %s\n", lines[printed].Message)
			}
		}

		dep, _, err := c.Functions.GetDeployment(ctx, ref, deploymentID)
		if err != nil {
			return err
		}
		switch dep.Status {
		case "ready":
			fn, _, err := c.Functions.Get(ctx, ref)
			if err != nil {
				return err
			}
			fmt.Printf("\n✔ Live: %s (rev-%d)\n", fn.URL, fn.CurrentRevisionNumber)
			return nil
		case "failed":
			if dep.Error != "" {
				return fmt.Errorf("deployment failed: %s", dep.Error)
			}
			return fmt.Errorf("deployment failed — see `raff functions get %s` or the dashboard", ref)
		}
		time.Sleep(2 * time.Second)
	}
}
