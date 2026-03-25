package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mogglemoss/fathom/config"
	"github.com/mogglemoss/fathom/model"
	"github.com/mogglemoss/fathom/server"
	"github.com/mogglemoss/fathom/ui"
)

// version is set at build time by GoReleaser via -ldflags.
var version = "dev"

var farewells = [][2]string{
	{"fathom has withdrawn from the waterline.", "the tide carries on without witness."},
	{"depth sounding concluded.", "the sea remains indifferent."},
	{"tidal observation suspended.", "the water does not notice."},
	{"fathom surfaces. the ocean does not pause.", "it never does."},
	{"session dissolved like salt in water.", "the levels continue their patient work."},
	{"monitoring ceased. the tide keeps its own counsel.", "as it has for millennia."},
	{"fathom has departed the intertidal zone.", "the moon still pulls."},
	{"signal lost to the depths.", "the predictions carry on."},
	{"dashboard offline. the harbor remains.", "the boats know what to do."},
	{"fathom folds. somewhere, a buoy bobs on.", "unmarked. unconcerned."},
}

func main() {
	stationFlag := flag.String("station", "", "NOAA station ID (overrides config)")
	themeFlag := flag.String("theme", "", "color theme: default, catppuccin, dracula, nord")
	verFlag := flag.Bool("version", false, "print version and exit")
	serveFlag := flag.Bool("serve", false, "serve the TUI over SSH using Wish")
	portFlag := flag.Int("port", 23234, "SSH server port (used with --serve)")
	hostFlag := flag.String("host", "0.0.0.0", "SSH server bind address (used with --serve)")
	flag.Parse()

	if *verFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if *stationFlag != "" {
		cfg.StationID = *stationFlag
	}

	// Theme priority: --theme flag > omarchy (auto-detected in initStyles) > config > default
	if *themeFlag != "" {
		ui.SetTheme(*themeFlag)
	} else if cfg.Theme != "" && cfg.Theme != "default" {
		ui.SetTheme(cfg.Theme)
	}

	if *serveFlag {
		if err := server.Start(*hostFlag, *portFlag, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	m := model.New(cfg)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	msg := farewells[rand.Intn(len(farewells))]
	fmt.Println()
	fmt.Println("  ◈  " + msg[0])
	fmt.Println("     " + msg[1])
	fmt.Println()
}
