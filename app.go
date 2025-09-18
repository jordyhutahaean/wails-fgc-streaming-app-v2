package main

import (
	"encoding/json"
	"os"
)

type App struct {
	scoreboard Scoreboard
	commentary Commentary
}

func NewApp() *App {
	return &App{
		scoreboard: Scoreboard{
			Game:      "sf",
			Style:     "minimalist",
			Titlecard: "",
			Visible1:  true,
			Visible2:  true,
			Visible3:  false,
			Visible4:  false,
		},
		commentary: Commentary{
			Commentator1: "",
			Description1: "",
			Commentator2: "",
			Description2: "",
			Visible:      false,
		},
	}
}

// -------------------- SCOREBOARD --------------------

type Scoreboard struct {
	Game      string `json:"game"`
	Style     string `json:"style"`
	Titlecard string `json:"titlecard"`

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
	Score3      int    `json:"score3"`
	Visible3    bool   `json:"visible3"`

	Player4     string `json:"player4"`
	Team4       string `json:"team4"`
	Controller4 string `json:"controller4"`
	Score4      int    `json:"score4"`
	Visible4    bool   `json:"visible4"`
}

func (a *App) GetScoreboard() Scoreboard {
	return a.scoreboard
}

func (a *App) SaveScoreboardJSON(data Scoreboard) error {
	a.scoreboard = data // keep in memory
	return saveJSON("scoreboard.json", data)
}

// -------------------- COMMENTARY --------------------

type Commentary struct {
	Commentator1 string `json:"commentator1"`
	Description1 string `json:"description1"`
	Commentator2 string `json:"commentator2"`
	Description2 string `json:"description2"`
	Visible      bool   `json:"visible"`
}

func (a *App) GetCommentary() Commentary {
	return a.commentary
}

func (a *App) SaveCommentaryJSON(data Commentary) error {
	file, err := os.Create("commentary.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// -------------------- UTILITY --------------------

func saveJSON(filename string, v interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
