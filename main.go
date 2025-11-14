package main

import (
	"embed"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/*
var assets embed.FS

// ------------------------------------------------------------
// FIX #2 — Cross-platform resource directory resolver
// ------------------------------------------------------------
func resourcePath(subpath string) string {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)

	// macOS .app bundle: binary is in Contents/MacOS
	macPath := filepath.Join(base, "..", "Resources", subpath)
	if _, err := os.Stat(macPath); err == nil {
		return macPath
	}

	// Windows/Linux
	return filepath.Join(base, subpath)
}

// ------------------------------------------------------------

func main() {
	app := NewApp()

	// Start lightweight webserver in background
	go func() {
		mux := http.NewServeMux()

		// Serve sponsors folder
		sponsorDir := resourcePath("sponsors")
		mux.Handle("/sponsors/", http.StripPrefix("/sponsors/",
			http.FileServer(http.Dir(sponsorDir)),
		))

		// Serve frontend assets
		overlayDir := resourcePath("frontend")
		os.MkdirAll(overlayDir, 0755)
		fs := http.FileServer(http.Dir(overlayDir))
		mux.Handle("/", fs)
		mux.Handle("/frontend/", http.StripPrefix("/frontend/", fs))

		mux.HandleFunc("/api/save-scoreboard", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var data map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			jsonPath := filepath.Join(resourcePath("frontend"), "scoreboard.json")

			out, _ := json.MarshalIndent(data, "", "  ")
			if err := os.WriteFile(jsonPath, out, 0644); err != nil {
				http.Error(w, "Save failed", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

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

	go startOverlayServer()
}

func startOverlayServer() {
	mux := http.NewServeMux()

	sponsorDir := resourcePath("sponsors")
	mux.Handle("/sponsors/", http.StripPrefix("/sponsors/",
		http.FileServer(http.Dir(sponsorDir)),
	))

	overlayDir := resourcePath("frontend")
	os.MkdirAll(overlayDir, 0755)
	fs := http.FileServer(http.Dir(overlayDir))
	mux.Handle("/", fs)
	mux.Handle("/frontend/", http.StripPrefix("/frontend/", fs))

	mux.HandleFunc("/api/save-scoreboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		jsonPath := filepath.Join(resourcePath("frontend"), "scoreboard.json")

		out, _ := json.MarshalIndent(data, "", "  ")
		if err := os.WriteFile(jsonPath, out, 0644); err != nil {
			http.Error(w, "Save failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	ports := []string{":34115", ":34116", ":34117"}
	for _, p := range ports {
		go func(port string) {
			if err := http.ListenAndServe(port, mux); err != nil {
				println("❌ Port", port, "failed:", err.Error())
			}
		}(p)
	}
}
