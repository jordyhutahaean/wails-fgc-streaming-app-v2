package main

import (
	"embed"
	"net/http"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/*
var assets embed.FS

func main() {
	app := NewApp()

	// Sponsor folder (relative to .exe location)
	sponsorDir := filepath.Join(".", "sponsors")

	// Serve files from sponsorDir at /sponsors/*
	http.Handle("/sponsors/",
		http.StripPrefix("/sponsors/",
			http.FileServer(http.Dir(sponsorDir)),
		),
	)

	err := wails.Run(&options.App{
		Title:  "Fighting Game Scoreboard Control",
		Width:  900,
		Height: 700,
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
