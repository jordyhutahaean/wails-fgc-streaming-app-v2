package main

import (
	"encoding/json"
	"os"
)

type App struct {
	scoreboard Scoreboard
}

func NewApp() *App {
	return &App{
		scoreboard: Scoreboard{
			Game:     "sf",
			Visible1: true,
			Visible2: true,
			Visible3: false,
			Visible4: false,
		},
	}
}

type Scoreboard struct {
	Game        string `json:"game"`
	Player1     string `json:"player1"`
	Team1       string `json:"team1"`
	Controller1 string `json:"controller1"`
	Score1      int    `json:"score1"`
	Visible1    bool   `json:"visible1"`

	Player2     string `json:"player2"`
	Team2       string `json:"team2"`
	Controller2 string `json:"controller2"`
	Score2      int    `json:"score2"`
	Visible2    bool   `json:"visible2"`

	Player3     string `json:"player3"`
	Team3       string `json:"team3"`
	Controller3 string `json:"controller3"`
	Visible3    bool   `json:"visible3"`

	Player4     string `json:"player4"`
	Team4       string `json:"team4"`
	Controller4 string `json:"controller4"`
	Visible4    bool   `json:"visible4"`
}

func (a *App) GetScoreboard() Scoreboard {
	return a.scoreboard
}

func (a *App) UpdateScoreboard(data Scoreboard) {
	a.scoreboard = data
}

func (a *App) SaveScoreboardJSON(data Scoreboard) error {
	file, err := os.Create("scoreboard.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
