package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/glueops/apiary/internal/api"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Code generation tools (migrations, SDKs)",
}

var generateMigrationsCmd = &cobra.Command{
	Use:   "migrations [name]",
	Short: "Generate Goose migrations from GORM models",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		log.Printf("Generating Control DB migration: %s", name)

		atlasControl := exec.Command("atlas", "migrate", "diff", name, "--env", "control")
		atlasControl.Stdout = os.Stdout
		atlasControl.Stderr = os.Stderr
		if err := atlasControl.Run(); err != nil {
			return fmt.Errorf("failed to generate control migrations: %w", err)
		}

		log.Printf("Generating Tenant DB migration: %s", name)

		atlasTenant := exec.Command("atlas", "migrate", "diff", name, "--env", "tenant")
		atlasTenant.Stdout = os.Stdout
		atlasTenant.Stderr = os.Stderr
		if err := atlasTenant.Run(); err != nil {
			return fmt.Errorf("failed to generate tenant migrations: %w", err)
		}

		log.Println("Migrations generated successfully.")
		return nil
	},
}

var generateSdkCmd = &cobra.Command{
	Use:   "sdk",
	Short: "Generate OpenAPI spec and TypeScript/Go SDKs",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("0. Removing existing generated SDKs...")
		for _, dir := range []string{
			"ui/src/sdk",
			"sdk/ts",
			"sdk/go",
		} {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to remove %s: %w", dir, err)
			}
		}

		log.Println("1. Extracting OpenAPI Spec from Huma...")

		cfg, err := config.Load()
		if err != nil {
			log.Printf("config.Load() failed (%v); using defaults for spec generation", err)
			cfg = &config.Config{
				BindHost:          "127.0.0.1",
				BindPort:          "8080",
				BasePath:          "/api/v1",
				BaseURL:           "http://localhost:8080",
				FrontendMode:      "none",
				ViteDevURL:        "http://localhost:5173",
				DatabaseURL:       "postgres://user:pass@localhost:5432/db?sslmode=disable",
				Env:               "development",
				JWTSecret:         "spec-only-secret",
				JWTIssuer:         "template",
				JWTAccessTTLMin:   60,
				JWTRefreshTTLHour: 720,
			}
		}
		cfg.FrontendMode = "none"

		if strings.TrimSpace(cfg.JWTSecret) == "" {
			cfg.JWTSecret = "spec-only-secret"
		}
		if strings.TrimSpace(cfg.JWTIssuer) == "" {
			cfg.JWTIssuer = "template"
		}
		if cfg.JWTAccessTTLMin <= 0 {
			cfg.JWTAccessTTLMin = 60
		}
		if cfg.JWTRefreshTTLHour <= 0 {
			cfg.JWTRefreshTTLHour = 720
		}

		dummyClient, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
			Workers: river.NewWorkers(),
		})
		if err != nil {
			return fmt.Errorf("failed to create dummy river client: %w", err)
		}

		dummyTokenSvc, _ := service.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, 0, 0)
		dummySessionSvc := service.NewTokenSessionService(dummyTokenSvc, nil, nil, nil)

		apiApp, err := api.Build(*cfg, &app.Deps{
			Config:        cfg,
			Jobs:          &workflow.MockEnqueuer{},
			River:         dummyClient,
			TokenSessions: dummySessionSvc,
		})
		if err != nil {
			return fmt.Errorf("api build: %w", err)
		}

		b, err := apiApp.API.OpenAPI().MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal openapi json: %w", err)
		}

		if err := os.MkdirAll("docs", 0o755); err != nil {
			return err
		}
		if err := os.WriteFile("docs/openapi.json", b, 0o644); err != nil {
			return fmt.Errorf("failed to write docs/openapi.json: %w", err)
		}

		specVersion := "0.0.0"
		{
			var doc struct {
				Info struct {
					Version string `json:"version"`
				} `json:"info"`
			}
			if json.Unmarshal(b, &doc) == nil && strings.TrimSpace(doc.Info.Version) != "" {
				specVersion = strings.TrimPrefix(strings.TrimSpace(doc.Info.Version), "v")
			}
		}

		log.Println("2. Generating embedded TypeScript SDK for UI with Hey API (ui/src/sdk)...")

		if err := generateTSUIHey(); err != nil {
			return err
		}

		log.Println("3. Generating standalone TypeScript SDK with OpenAPI Generator (sdk/ts)...")
		if err := generateTSPackagedOAG("docs/openapi.json", "sdk/ts", []string{
			"--additional-properties=supportsES6=true,npmName=@glueops/apiary-sdk,npmVersion=" + specVersion,
		}); err != nil {
			return err
		}

		log.Println("4. Generating standalone Go SDK (sdk/go)...")
		if err := generateGo("docs/openapi.json", "sdk/go", specVersion); err != nil {
			return err
		}

		log.Println("SDKs generated successfully.")
		return nil
	},
}

// goSDKModulePath is the import path consumers use. It must equal the SDK's
// directory within this repository, or the nested module is unfetchable.
const goSDKModulePath = "github.com/glueops/apiary/sdk/go"

// goSDKPackageName is both the Go package name and the last path element
// openapi-generator appends when it composes the module path. Kept in one place
// so the rewrite below and the --additional-properties flag cannot disagree.
const goSDKPackageName = "apiary"

func generateGo(inputSpec, outDir, version string) error {
	_ = os.RemoveAll(outDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	gitUserID := "glueops"
	gitRepoID := filepath.ToSlash(filepath.Join("apiary", "sdk", "go"))

	oagBin, err := localNodeBin("openapi-generator-cli")
	if err != nil {
		return err
	}

	args := []string{
		"generate",
		"-i", inputSpec,
		"-g", "go",
		"-o", outDir,

		"--additional-properties=packageName=" + goSDKPackageName + ",packageVersion=" + version + ",withGoMod=true,isGoSubmodule=true",
		"--git-user-id", gitUserID,
		"--git-repo-id", gitRepoID,
	}
	if err := run(exec.Command(oagBin, args...), "go -> "+outDir); err != nil {
		return err
	}

	if err := patchGoClientLogging(outDir); err != nil {
		return err
	}

	edit := exec.Command("go", "mod", "edit", "-go=1.26.1")
	edit.Dir = outDir
	if err := run(edit, "set go version -> "+outDir); err != nil {
		return err
	}

	// Correct the module path the generator produced.
	//
	// openapi-generator composes it as <git-user-id>/<git-repo-id>/<packageName>,
	// which appends the Go PACKAGE name to the path — here yielding
	// .../sdk/go/apiary while the code is written to sdk/go. For a module nested
	// inside a repository, Go resolves the path as repository + subdirectory, so
	// a module path that does not equal its own directory cannot be fetched at
	// all: `go get` looks for a sdk/go/apiary directory that does not exist.
	//
	// That is not hypothetical — it is why agent/ could never build against this
	// SDK. The package stays "apiary"; only the module path changes, and a
	// package name differing from the last path element is ordinary for an SDK.
	modEdit := exec.Command("go", "mod", "edit", "-module", goSDKModulePath)
	modEdit.Dir = outDir
	if err := run(modEdit, "set module path -> "+outDir); err != nil {
		return err
	}

	// The module path in go.mod is only half the correction: the generator also
	// wrote that composed path into the imports of every file it produced,
	// including the generated test package. Left alone, `go mod tidy` below tries
	// to resolve github.com/glueops/apiary/sdk/go/apiary over the network and
	// fails — the path names a directory that does not exist in the repository.
	generated := "github.com/" + gitUserID + "/" + gitRepoID + "/" + goSDKPackageName
	if err := rewriteImportPath(outDir, generated, goSDKModulePath); err != nil {
		return err
	}

	tidy := exec.Command("go", "mod", "tidy", "-go=1.26.1")
	tidy.Dir = outDir
	if err := run(tidy, "go mod tidy -> "+outDir); err != nil {
		return err
	}

	return nil
}

func patchGoClientLogging(outDir string) error {
	clientFile := filepath.Join(outDir, "client.go")
	b, err := os.ReadFile(clientFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", clientFile, err)
	}

	content := string(b)

	unsafeRequestDump := "\tif c.cfg.Debug {\n\t\tdump, err := httputil.DumpRequestOut(request, true)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\tlog.Printf(\"\\n%s\\n\", string(dump))\n\t}"

	unsafeResponseDump := "\tif c.cfg.Debug {\n\t\tdump, err := httputil.DumpResponse(resp, true)\n\t\tif err != nil {\n\t\t\tresp.Body.Close()\n\t\t\treturn nil, err\n\t\t}\n\t\tlog.Printf(\"\\n%s\\n\", string(dump))\n\t}"

	safeRequestLog := "\tif c.cfg.Debug {\n\t\t// Log request metadata only (method, URL) to avoid logging sensitive data in headers or body.\n\t\tlog.Printf(\"HTTP request: %s %s\", request.Method, request.URL.String())\n\t}"
	safeResponseLog := "\tif c.cfg.Debug {\n\t\t// Log response metadata only (status) to avoid logging sensitive data in headers or body.\n\t\tlog.Printf(\"HTTP response: %s\", resp.Status)\n\t}"

	patched := strings.ReplaceAll(content, unsafeRequestDump, safeRequestLog)
	patched = strings.ReplaceAll(patched, unsafeResponseDump, safeResponseLog)

	if patched == content {

		if strings.Contains(content, "request.Method, request.URL.String()") {
			log.Printf("note: %s already has metadata-only logging; skipping patch", clientFile)
		} else {
			log.Printf("WARNING: %s does not contain the expected unsafe logging patterns; the generator template may have changed - review debug logging in callAPI for sensitive data exposure", clientFile)
		}
		return nil
	}

	if err := os.WriteFile(clientFile, []byte(patched), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", clientFile, err)
	}
	log.Printf("patched %s: replaced unsafe request/response dumps with metadata-only logging", clientFile)
	return nil
}

func run(c *exec.Cmd, label string) error {
	log.Printf("running: %s: %s %s", label, c.Path, strings.Join(c.Args[1:], " "))
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", label, err)
	}
	return nil
}

var generateTerraformCmd = &cobra.Command{
	Use:   "terraform",
	Short: "Generate Terraform provider artifacts",
}

var generateTerraformHashiCmd = &cobra.Command{
	Use:   "hashicorp",
	Short: "Generate Terraform provider code via HashiCorp codegen",
	RunE: func(cmd *cobra.Command, args []string) error {
		specPath := "docs/openapi.json"
		cfgPath := "terraform/generator_config.yml"
		specOut := "terraform/provider_code_spec.json"
		outDir := "terraform-provider-apiary/internal/provider"

		if err := generateOpenAPISpec(specPath); err != nil {
			return err
		}

		for _, dir := range []string{
			specOut,
			outDir,
		} {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("failed to remove %s: %w", dir, err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(specOut), 0o755); err != nil {
			return err
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}

		openapiCmd := exec.Command(
			"tfplugingen-openapi",
			"generate",
			"--config", cfgPath,
			"--output", specOut,
			specPath,
		)
		if err := run(openapiCmd, "terraform hashicorp openapi -> "+specOut); err != nil {
			return err
		}

		frameworkCmd := exec.Command(
			"tfplugingen-framework",
			"generate",
			"all",
			"--input", specOut,
			"--output", outDir,
		)
		if err := run(frameworkCmd, "terraform hashicorp framework -> "+outDir); err != nil {
			return err
		}

		log.Println("HashiCorp Terraform provider artifacts generated successfully.")
		return nil
	},
}

var generateTerraformOAGCmd = &cobra.Command{
	Use:   "openapi-generator",
	Short: "Generate Terraform provider via OpenAPI Generator",
	RunE: func(cmd *cobra.Command, args []string) error {
		specPath := "docs/openapi.json"
		outDir := "terraform-provider-apiary"

		if err := generateOpenAPISpec(specPath); err != nil {
			return err
		}

		if err := os.RemoveAll(outDir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", outDir, err)
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}

		oagBin, err := localNodeBin("openapi-generator-cli")
		if err != nil {
			return err
		}

		genArgs := []string{
			"generate",
			"-i", specPath,
			"-g", "terraform-provider",
			"-o", outDir,
			"--additional-properties=providerName=apiary,packageName=apiary,providerAddress=registry.terraform.io/glueops/apiary",
			"--git-user-id", "glueops",
			"--git-repo-id", "terraform-provider-apiary",
		}
		if err := run(exec.Command(oagBin, genArgs...), "terraform-provider -> "+outDir); err != nil {
			return err
		}

		log.Println("OpenAPI Generator Terraform provider generated successfully.")
		return nil
	},
}

func generateTSUIHey() error {
	outDir := "ui/src/sdk"

	_ = os.RemoveAll(outDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	bin, err := localNodeBin("openapi-ts")
	if err != nil {
		return err
	}

	if err := run(exec.Command(bin), "hey-api ui sdk -> "+outDir); err != nil {
		return err
	}

	return nil
}

// localNodeBin resolves a generator from the repo's own node_modules rather
// than fetching one at run time.
//
// This previously ran `npx --yes @hey-api/openapi-ts@0.93.0`, which was wrong in
// two compounding ways. It pinned 0.93.0 while the root package.json pinned
// 0.99.0, so the declared dependency was never the one that ran; and `--yes`
// resolved the generator's UNBOUNDED peer dependency (`typescript: >=5.5.3`) to
// whatever was newest — which is now TypeScript 7, the native port, whose JS API
// deliberately drops the legacy compiler surface. hey-api then read
// `ts.NewLineKind.LineFeed` off undefined and the whole SDK build died with a
// TypeError that says nothing about versions.
//
// Resolving locally makes the toolchain reproducible: the generator and its
// TypeScript both come from the lockfile, so the SDK generated today is the SDK
// generated next month from the same spec.
func localNodeBin(name string) (string, error) {
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	bin := filepath.Join("node_modules", ".bin", name)

	if _, err := os.Stat(bin); err != nil {
		// Deliberately NOT falling back to npx. A fallback would restore the
		// exact failure above, silently, on any machine that had not installed
		// dependencies — and it would look like a generator bug rather than a
		// missing install.
		return "", fmt.Errorf(
			"%s is not installed: run the package manager at the repo root first "+
				"(node_modules/.bin/%s is missing)", name, name)
	}

	abs, err := filepath.Abs(bin)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func generateTSPackagedOAG(inputSpec, outDir string, extraArgs []string) error {
	_ = os.RemoveAll(outDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	gitUserID := "glueops"
	gitRepoID := "apiary"

	oagBin, err := localNodeBin("openapi-generator-cli")
	if err != nil {
		return err
	}

	args := []string{
		"generate",
		"-i", inputSpec,
		"-g", "typescript-fetch",
		"-o", outDir,
		"--git-user-id", gitUserID,
		"--git-repo-id", gitRepoID,
	}
	args = append(args, extraArgs...)

	return run(exec.Command(oagBin, args...), "typescript-fetch -> "+outDir)
}

func generateOpenAPISpec(outPath string) error {
	log.Println("Extracting OpenAPI Spec from Huma...")

	cfg, err := config.Load()
	if err != nil {
		log.Printf("config.Load() failed (%v); using defaults for spec generation", err)
		cfg = &config.Config{
			BindHost:          "127.0.0.1",
			BindPort:          "8080",
			BasePath:          "/api/v1",
			BaseURL:           "http://localhost:8080",
			FrontendMode:      "none",
			ViteDevURL:        "http://localhost:5173",
			DatabaseURL:       "postgres://user:pass@localhost:5432/db?sslmode=disable",
			Env:               "development",
			JWTSecret:         "spec-only-secret",
			JWTIssuer:         "nexus",
			JWTAccessTTLMin:   60,
			JWTRefreshTTLHour: 720,
		}
	}
	cfg.FrontendMode = "none"

	if strings.TrimSpace(cfg.JWTSecret) == "" {
		cfg.JWTSecret = "spec-only-secret"
	}
	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		cfg.JWTIssuer = "nexus"
	}
	if cfg.JWTAccessTTLMin <= 0 {
		cfg.JWTAccessTTLMin = 60
	}
	if cfg.JWTRefreshTTLHour <= 0 {
		cfg.JWTRefreshTTLHour = 720
	}

	dummyClient, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Workers: river.NewWorkers(),
	})
	if err != nil {
		return fmt.Errorf("failed to create dummy river client: %w", err)
	}

	dummyTokenSvc2, _ := service.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, 0, 0)
	dummySessionSvc2 := service.NewTokenSessionService(dummyTokenSvc2, nil, nil, nil)

	apiApp, err := api.Build(*cfg, &app.Deps{
		Config:        cfg,
		Jobs:          &workflow.MockEnqueuer{},
		River:         dummyClient,
		TokenSessions: dummySessionSvc2,
	})
	if err != nil {
		return fmt.Errorf("api build: %w", err)
	}

	b, err := apiApp.API.OpenAPI().MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal openapi json: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	return nil
}

func init() {
	generateTerraformCmd.AddCommand(generateTerraformHashiCmd)
	generateTerraformCmd.AddCommand(generateTerraformOAGCmd)

	generateCmd.AddCommand(generateMigrationsCmd)
	generateCmd.AddCommand(generateSdkCmd)
	generateCmd.AddCommand(generateTerraformCmd)

	rootCmd.AddCommand(generateCmd)
}

// rewriteImportPath replaces one import path with another across generated Go
// sources. It is a plain string substitution because the input is generator
// output, not hand-written code: the path appears only in import statements and
// there is no risk of matching prose.
func rewriteImportPath(dir, from, to string) error {
	if from == to {
		return nil
	}

	// The walk only COLLECTS paths; every filesystem operation happens after it
	// returns. Reading and writing inside a WalkDir callback acts on a path the
	// walk resolved earlier, which is a symlink TOCTOU the linter is right to
	// flag — and here it would let a crafted link in generator output redirect a
	// write anywhere the process can reach.
	var files []string
	if err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Regular files only: a symlink is not something to rewrite in place.
		if d.IsDir() || !d.Type().IsRegular() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		files = append(files, p)
		return nil
	}); err != nil {
		return err
	}

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if !bytes.Contains(b, []byte(from)) {
			continue
		}
		if err := os.WriteFile(f, bytes.ReplaceAll(b, []byte(from), []byte(to)), 0o644); err != nil {
			return err
		}
	}
	return nil
}
