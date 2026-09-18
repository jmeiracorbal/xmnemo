package cmd

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	adapters  []string
	mnemoVer  string
	outputBin string
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a custom mnemo binary with the specified adapters",
	Long: `Generates a temporary Go workspace with the specified adapters
blank-imported alongside the mnemo core, then compiles it and places
the resulting binary at the output path (default: ~/.local/bin/mnemo).

Each --adapter flag accepts a fully-qualified Go module path with an
optional version suffix (e.g. github.com/acme/mnemo-zed@v1.2.0).
Omitting the version resolves to the latest published version.

The built-in mnemo adapters (Claude Code, Codex, Cursor, OpenCode, Pi)
are included automatically unless --no-builtins is passed.`,
	Example: `  xmnemo build --adapter github.com/acme/mnemo-zed@v1.2.0
  xmnemo build --adapter github.com/acme/mnemo-zed --adapter github.com/corp/mnemo-internal
  xmnemo build --adapter github.com/acme/mnemo-zed --no-builtins --output ~/bin/mnemo`,
	RunE: runBuild,
}

var noBuiltins bool

func init() {
	buildCmd.Flags().StringArrayVar(&adapters, "adapter", nil, "adapter module to include (repeatable; module[@version])")
	buildCmd.Flags().StringVar(&mnemoVer, "mnemo-version", "latest", "mnemo core version to build against")
	buildCmd.Flags().StringVar(&outputBin, "output", "", "output binary path (default: ~/.local/bin/mnemo)")
	buildCmd.Flags().BoolVar(&noBuiltins, "no-builtins", false, "exclude built-in adapters (Claude Code, Codex, Cursor, OpenCode, Pi)")
	_ = buildCmd.MarkFlagRequired("adapter")
	rootCmd.AddCommand(buildCmd)
}

const goModTmpl = `module xmnemo_build

go 1.26.6

require github.com/jmeiracorbal/mnemo {{.MnemoVersion}}
{{range .Adapters}}require {{.Module}} {{.Version}}
{{end}}
`

type adapterSpec struct {
	Module  string
	Version string
}

func parseAdapter(s string) adapterSpec {
	parts := strings.SplitN(s, "@", 2)
	spec := adapterSpec{Module: parts[0]}
	if len(parts) == 2 {
		spec.Version = parts[1]
	} else {
		spec.Version = "latest"
	}
	return spec
}

func resolveOutput() (string, error) {
	if outputBin != "" {
		return outputBin, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "bin", "mnemo"), nil
}

func runBuild(cmd *cobra.Command, _ []string) error {
	if len(adapters) == 0 {
		return fmt.Errorf("at least one --adapter is required")
	}

	specs := make([]adapterSpec, 0, len(adapters))
	for _, a := range adapters {
		specs = append(specs, parseAdapter(a))
	}

	outPath, err := resolveOutput()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "xmnemo-build-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd.Printf("Building custom mnemo binary...\n")
	cmd.Printf("  mnemo version : %s\n", mnemoVer)
	cmd.Printf("  adapters      : %v\n", adapters)
	cmd.Printf("  output        : %s\n", outPath)

	if err := writeGoMod(tmpDir, specs); err != nil {
		return err
	}
	if err := writeMain(tmpDir, specs); err != nil {
		return err
	}

	if err := goGet(tmpDir, specs); err != nil {
		return err
	}
	if err := goBuild(tmpDir, outPath); err != nil {
		return err
	}

	cmd.Printf("Done. Custom mnemo binary installed at %s\n", outPath)
	return nil
}

func writeGoMod(dir string, specs []adapterSpec) error {
	tmpl, err := template.New("gomod").Parse(goModTmpl)
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "go.mod"))
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, map[string]any{
		"MnemoVersion": mnemoVer,
		"Adapters":     specs,
	})
}

func writeMain(dir string, specs []adapterSpec) error {
	var sb strings.Builder
	sb.WriteString("package main\n\nimport (\n")
	sb.WriteString("\t_ \"github.com/jmeiracorbal/mnemo/cmd\"\n")
	if !noBuiltins {
		for _, builtin := range []string{
			"github.com/jmeiracorbal/mnemo-adapters/mnemo-claude",
			"github.com/jmeiracorbal/mnemo-adapters/mnemo-codex",
			"github.com/jmeiracorbal/mnemo-adapters/mnemo-cursor",
			"github.com/jmeiracorbal/mnemo-adapters/mnemo-opencode",
			"github.com/jmeiracorbal/mnemo-adapters/mnemo-pi",
		} {
			fmt.Fprintf(&sb, "\t_ %q\n", builtin)
		}
	}
	for _, s := range specs {
		fmt.Fprintf(&sb, "\t_ %q\n", s.Module)
	}
	sb.WriteString(")\n\nfunc main() {}\n")

	src, err := format.Source([]byte(sb.String()))
	if err != nil {
		return fmt.Errorf("format generated main.go: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), src, 0o644)
}

func goGet(dir string, specs []adapterSpec) error {
	args := []string{"get"}
	for _, s := range specs {
		if s.Version == "latest" {
			args = append(args, s.Module+"@latest")
		} else {
			args = append(args, s.Module+"@"+s.Version)
		}
	}
	return runGoCmd(dir, args...)
}

func goBuild(dir, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	return runGoCmd(dir, "build", "-o", outPath, ".")
}

func runGoCmd(dir string, args ...string) error {
	c := exec.Command("go", args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
