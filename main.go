package main

import (
	"embed"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	app := NewApp()

	// Start lightweight webserver in background
	go func() {
		mux := http.NewServeMux()

		// Serve sponsors folder (resolve path next to binary)
		exe, _ := os.Executable()
		base := filepath.Dir(exe)
		sponsorDir := filepath.Join(base, "frontend", "sponsors")

		mux.Handle("/sponsors/", http.StripPrefix("/sponsors/",
			http.FileServer(http.Dir(sponsorDir)),
		))

		// Serve overlays (scoreboard.html, single.html, double.html, commentary.html)
		overlayDir := filepath.Join(".", "frontend")
		mux.Handle("/", http.FileServer(http.Dir(overlayDir)))

		// Listen on port 34115
		if err := http.ListenAndServe(":34115", mux); err != nil {
			println("Overlay server error:", err.Error())
		}
	}()

	// Run Wails app
	err := wails.Run(&options.App{
		Title:  "Fighting Game Scoreboard Control",
		Width:  860,
		Height: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
