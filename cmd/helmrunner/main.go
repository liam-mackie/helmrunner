package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"github.com/liam-mackie/helmrunner/internal/helm"
	"github.com/liam-mackie/helmrunner/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	templateMode := flag.Bool("template", false, "render templates to stdout instead of installing")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	defs, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(defs) == 0 {
		fmt.Fprintf(os.Stderr, "No definitions found in %s\n", dir)
		os.Exit(1)
	}

	model := tui.New(defs, *templateMode)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result := finalModel.(tui.Model).GetResult()
	if result.Aborted {
		os.Exit(0)
	}

	if *templateMode {
		var outputs []string
		for _, def := range result.Definitions {
			rendered, err := helm.Template(context.Background(), def)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error templating %s: %v\n", def.Name, err)
				os.Exit(1)
			}
			outputs = append(outputs, rendered)
		}
		fmt.Print(strings.Join(outputs, "\n---\n"))
	}
}
