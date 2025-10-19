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

		// Resolve path next to binary
		exe, _ := os.Executable()
		base := filepath.Dir(exe)

		// ✅ Serve bin/sponsors/ at /sponsors/
		sponsorDir := filepath.Join(base, "sponsors")
		mux.Handle("/sponsors/", http.StripPrefix("/sponsors/",
			http.FileServer(http.Dir(sponsorDir)),
		))

		// --- Serve frontend folder ---
		overlayDir := filepath.Join(base, "frontend")
		fs := http.FileServer(http.Dir(overlayDir))
		mux.Handle("/", fs)
		mux.Handle("/frontend/", http.StripPrefix("/frontend/", fs)) // optional: supports both
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
