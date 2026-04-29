package main

import (
	"bytes"
	"embed"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	// add these to your existing imports
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
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
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

		mux.HandleFunc("/api/load-players", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.Method != "GET" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			filePath := filepath.Join(resourcePath("frontend"), "data.csv")
			f, err := os.Open(filePath)
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
			defer f.Close()

			csvReader := csv.NewReader(f)
			records, err := csvReader.ReadAll()
			if err != nil || len(records) < 2 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]map[string]string{})
				return
			}

			var players []map[string]string
			for _, row := range records[1:] {
				if len(row) >= 2 {
					players = append(players, map[string]string{
						"name": strings.TrimSpace(row[0]),
						"team": strings.TrimSpace(row[1]),
					})
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(players)
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
		OnStartup: app.startup,
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

	mux.HandleFunc("/api/debug-sets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Re-run the raw query and return whatever start.gg sends back
		query := `query EventSets($slug: String!, $page: Int!) {
  event(slug: $slug) {
    name
    sets(perPage: 50, page: $page, filters: {hideEmpty: false}) {
      pageInfo { totalPages }
      nodes {
        id
        fullRoundText
        slots {
          entrant { name }
        }
      }
    }
  }
}`
		payload := map[string]interface{}{
			"query": query,
			"variables": map[string]interface{}{
				"slug": appState.LastEventSlug,
				"page": 1,
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "https://api.start.gg/gql/alpha", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if appState.LastAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+appState.LastAPIKey)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Forward raw response directly
		var raw interface{}
		json.NewDecoder(resp.Body).Decode(&raw)
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"slug":   appState.LastEventSlug,
			"apiKey": appState.LastAPIKey != "",
			"raw":    raw,
		})
	})

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

	mux.HandleFunc("/api/load-players", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filePath := filepath.Join(resourcePath("frontend"), "data.csv")

		f, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		csvReader := csv.NewReader(f)
		records, err := csvReader.ReadAll()
		if err != nil || len(records) < 2 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]string{})
			return
		}

		var players []map[string]string
		for _, row := range records[1:] {
			if len(row) >= 2 {
				players = append(players, map[string]string{
					"name": strings.TrimSpace(row[0]),
					"team": strings.TrimSpace(row[1]),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(players)
	})

	mux.HandleFunc("/api/get-sets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appState.LastSets)
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
