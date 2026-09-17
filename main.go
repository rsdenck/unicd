package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/denck/unicd/cmd"
)

const (
	ColorAzul  = "\033[38;5;39m"
	ColorVerde = "\033[38;5;191m"
	ColorReset = "\033[0m"
)

const banner = "\u00a7G\u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2557\n" +
	"\u00a7G\u2551\u00a7R                                                                                               \u00a7G\u2551\n" +
	"\u00a7G\u2551   \u00a7A _    _ _   _ _____ ______ _____ ____  _    _ ______    _____ _      ____  _    _ _____  \u00a7G   \u2551\n" +
	"\u00a7G\u2551   \u00a7A| |  | | \\ | |_   _|  ____|_   _/ __ \\| |  | |  ____|  / ____| |    / __ \\| |  | |  __ \\ \u00a7G   \u2551\n" +
	"\u00a7G\u2551   \u00a7A| |  | |  \\| | | | | |__    | || |  | | |  | | |__    | |    | |   | |  | | |  | | |  | |\u00a7G   \u2551\n" +
	"\u00a7G\u2551   \u00a7A| |  | | . ` | | | |  __|   | || |  | | |  | |  __|   | |    | |   | |  | | |  | | |  | |\u00a7G   \u2551\n" +
	"\u00a7G\u2551   \u00a7A| |__| | |\\  |_| |_| |     _| || |__| | |__| | |____  | |____| |___| |__| | |__| | |__| |\u00a7G   \u2551\n" +
	"\u00a7G\u2551   \u00a7A \\____/|_| \\_|_____|_|    |_____\\___\\_\\\\____/|______|  \\_____|______\\____/ \\____/|_____/ \u00a7G   \u2551\n" +
	"\u00a7G\u2551\u00a7R                                                                                               \u00a7G\u2551\n" +
	"\u00a7G\u2551\u00a7R                       UNIFIQUE CLOUD DIRECTOR  \u00b7  vCD  \u00b7  OBJECT STORAGE                      \u00a7G\u2551\n" +
	"\u00a7G\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u255d"

func initTerminal() {
	if runtime.GOOS == "windows" {
		os.Setenv("TERM", "xterm-256color")
	}
}

func printBanner() {
	initTerminal()
	b := strings.NewReplacer("§A", ColorAzul, "§G", ColorVerde, "§R", ColorReset).Replace(banner)
	fmt.Print(b)
	fmt.Println(ColorVerde + "\n >> Provider CLI ativo e pronto para execução." + ColorReset)
	fmt.Println()
}

func main() {
	showBanner := false
	noBanner := false
	args := os.Args[1:]
	for _, a := range args {
		if a == "--no-banner" || a == "--no-banner=true" {
			noBanner = true
		}
	}
	if !noBanner {
		if len(args) == 0 {
			showBanner = true
		} else {
			for _, a := range args {
				if a == "--help" || a == "-h" || a == "login" {
					showBanner = true
					break
				}
			}
		}
	}
	if showBanner {
		printBanner()
	}
	cmd.Execute()
}
