package main

import (
	"encoding/json"
	"os"
)

// App holds scoreboard, commentary and brackets
type App struct {
	scoreboard Scoreboard
	commentary Commentary
	bracket    Bracket
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
		bracket: NewEmptyBracket(),
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

// SaveCommentaryJSON saves the commentary data to disk
func (a *App) SaveCommentaryJSON(data map[string]interface{}) error {
	f, err := os.Create("commentary.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// -------------------- BRACKETS --------------------

// For persistence we store bracket JSON. Frontend performs progression logic for single-elim.
type Bracket struct {
	Single SingleBracket `json:"single"`
	Double DoubleBracket `json:"double"`
}

type SingleBracket struct {
	// Top 8 players (seeded order). Always length 8.
	Players []string `json:"players"`
	// Scores for matches (keyed)
	// allowed keys: qf1,qf2,qf3,qf4, sf1,sf2, f1
	Scores map[string][2]int `json:"scores"`
	// Resolved winners (optional cache)
	// Keys same as scores: qf1,qf2,..., f1
	Winners map[string]string `json:"winners"`
}

type DoubleBracket struct {
	// We'll store players and score buckets — frontend can implement complex flow later.
	Players []string          `json:"players"` // length 8
	Scores  map[string][2]int `json:"scores"`
	Winners map[string]string `json:"winners"`
	Meta    map[string]string `json:"meta"` // any helper data
}

func NewEmptyBracket() Bracket {
	return Bracket{
		Single: SingleBracket{
			Players: make([]string, 8),
			Scores:  make(map[string][2]int),
			Winners: make(map[string]string),
		},
		Double: DoubleBracket{
			Players: make([]string, 8),
			Scores:  make(map[string][2]int),
			Winners: make(map[string]string),
			Meta:    make(map[string]string),
		},
	}
}

func (a *App) GetBracket() Bracket {
	return a.bracket
}

func (a *App) SaveBracketJSON(data Bracket) error {
	a.bracket = data
	return saveJSON("bracket.json", data)
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
