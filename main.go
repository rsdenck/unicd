package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/denck/unicd/cmd"
)

const (
	ColorAzul  = "\033[38;5;18m"
	ColorVerde = "\033[38;5;191m"
	ColorReset = "\033[0m"
)

func initTerminal() {
	if runtime.GOOS == "windows" {
		os.Setenv("TERM", "xterm-256color")
	}
}

func printBanner() {
	initTerminal()
	fmt.Println(ColorVerde + "::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::" + ColorReset)
	fmt.Println(ColorVerde + "::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::" + ColorReset)
	fmt.Println(ColorVerde + ":: " + ColorAzul + "_   _       _  __ _                     ____ _                 _" + ColorVerde + " ::")
	fmt.Println(ColorVerde + "::" + ColorAzul + "| | | |_ __ (_)/ _(_) __ _ _   _  ___   / ___| | ___  _   _  __| |" + ColorVerde + "::")
	fmt.Println(ColorVerde + "::" + ColorAzul + "| | | | '_ \\| | |_| |/ _` | | | |/ _ \\ | |   | |/ _ \\| | | |/ _` |" + ColorVerde + "::")
	fmt.Println(ColorVerde + "::" + ColorAzul + "| |_| | | | | |  _| | (_| | |_| |  __/ | |___| | (_) | |_| | (_| |" + ColorVerde + "::")
	fmt.Println(ColorVerde + "::" + ColorAzul + " \\___/|_| |_|_|_| |_|\\__, |\\__,_|\\___|  \\____|_|\\___/ \\__,_|\\__,_|" + ColorVerde + "::")
	fmt.Println(ColorVerde + ":: " + ColorAzul + "                       |_|                                      " + ColorVerde + " ::")
	fmt.Println(ColorVerde + "::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::" + ColorReset)
	fmt.Println(ColorVerde + "::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::" + ColorReset)
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
