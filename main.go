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
		os.MkdirAll(overlayDir, 0755)
		fs := http.FileServer(http.Dir(overlayDir))
		mux.Handle("/", fs)
		mux.Handle("/frontend/", http.StripPrefix("/frontend/", fs))

		// --- Multi-port startup ---
		ports := []string{":34115", ":34116", ":34117"}

		for _, p := range ports {
			go func(port string) {
				println("🔌 Starting overlay server on port", port, "...")
				println("✅ Access via: http://localhost" + port + "/scoreboard.html")
				if err := http.ListenAndServe(port, mux); err != nil {
					println("❌ Port", port, "failed:", err.Error())
				}
			}(p)
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
