package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// App holds scoreboard, commentary, brackets and sponsors
type App struct {
	scoreboard Scoreboard
	commentary Commentary
	bracket    Bracket
	sponsorDir string
}

// -------------------- INIT --------------------

func NewApp() *App {
	// ✅ Always resolve path relative to the executable (works in wails build)
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	dir := filepath.Join(base, "sponsors")

	os.MkdirAll(dir, 0755)

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
		bracket:    NewEmptyBracket(),
		sponsorDir: dir,
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
	return saveJSON("./frontend/scoreboard.json", data)
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

func (a *App) SaveCommentaryJSON(data map[string]interface{}) error {
	f, err := os.Create("./frontend/commentary.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// -------------------- BRACKETS --------------------

type Bracket struct {
	Single SingleBracket `json:"single"`
	Double DoubleBracket `json:"double"`
}

type SingleBracket struct {
	Title   string            `json:"title"`
	Players []string          `json:"players"`
	Scores  map[string][2]int `json:"scores"`
	Winners map[string]string `json:"winners"`
}

type DoubleBracket struct {
	Title   string            `json:"title"`
	Players []string          `json:"players"`
	Scores  map[string][2]int `json:"scores"`
	Winners map[string]string `json:"winners"`
	Losers  map[string]string `json:"losers"`
	Meta    map[string]string `json:"meta"`
}

func NewEmptyBracket() Bracket {
	return Bracket{
		Single: SingleBracket{
			Title:   "Single Bracket",
			Players: make([]string, 8),
			Scores:  make(map[string][2]int),
			Winners: make(map[string]string),
		},
		Double: DoubleBracket{
			Title:   "Double Bracket",
			Players: make([]string, 8),
			Scores:  make(map[string][2]int),
			Winners: make(map[string]string),
			Losers:  make(map[string]string),
			Meta:    make(map[string]string),
		},
	}
}

func (a *App) GetBracket() Bracket {
	return a.bracket
}

func (a *App) SaveBracketJSON(data Bracket) error {
	a.bracket = data
	return saveJSON("./frontend/bracket.json", data)
}

// -------------------- SPONSORS --------------------

func (a *App) UploadSponsor(filename string, data []byte) error {
	dir := filepath.Join("frontend", "sponsors")
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	// Fix 1: Always add "sponsors/" prefix when updating JSON
	jsonPath := filepath.Join("frontend", "sponsors.json")
	var list []string
	if b, err := os.ReadFile(jsonPath); err == nil {
		json.Unmarshal(b, &list)
	}

	// Fix 2: Use prefixed path and avoid duplicates
	prefixedPath := "sponsors/" + filename
	for _, n := range list {
		if n == prefixedPath {
			goto SKIP_APPEND
		}
	}
	list = append(list, prefixedPath) // Add with sponsors/ prefix
SKIP_APPEND:
	out, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(jsonPath, out, 0644)
}

// SaveSponsor saves an uploaded file to sponsors folder and updates sponsors.json
func (a *App) SaveSponsor(name string, data []byte) error {
	target := filepath.Join(a.sponsorDir, filepath.Base(name))
	println("💾 Saving sponsor:", target)

	if err := os.WriteFile(target, data, 0644); err != nil {
		println("❌ SaveSponsor error:", err.Error())
		return err
	}

	return a.saveSponsorList()
}

// DeleteSponsor removes a file from sponsors folder and updates sponsors.json
func (a *App) DeleteSponsor(name string) error {
	target := filepath.Join(a.sponsorDir, filepath.Base(name))
	println("🗑️ Deleting sponsor:", target)
	_ = os.Remove(target)
	return a.saveSponsorList()
}

// GetSponsors returns list of sponsor image paths
func (a *App) GetSponsors() ([]string, error) {
	// Always use the sponsors.json first
	list, err := a.GetSponsorList()
	if err == nil && len(list) > 0 {
		return list, nil
	}

	// Fallback to directory scan
	files, err := os.ReadDir(a.sponsorDir)
	if err != nil {
		return nil, err
	}

	var sponsors []string
	for _, f := range files {
		if !f.IsDir() {
			sponsors = append(sponsors, "sponsors/"+f.Name())
		}
	}
	sort.Strings(sponsors)

	// Save the list for next time
	a.saveSponsorList()
	return sponsors, nil
}

// saveSponsorList writes a JSON file (next to exe → frontend/sponsors.json)
func (a *App) saveSponsorList() error {
	files, err := os.ReadDir(a.sponsorDir)
	if err != nil {
		return err
	}
	var list []string
	for _, f := range files {
		if !f.IsDir() {
			list = append(list, "sponsors/"+f.Name())
		}
	}
	sort.Strings(list)

	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	jsonPath := filepath.Join(base, "frontend", "sponsors.json")

	os.MkdirAll(filepath.Dir(jsonPath), 0755)
	println("📝 Writing sponsors.json to:", jsonPath)

	return saveJSON(jsonPath, list)
}

// GetSponsorList reads frontend/sponsors.json (next to exe)
func (a *App) GetSponsorList() ([]string, error) {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	jsonPath := filepath.Join(base, "frontend", "sponsors.json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}
	var sponsors []string
	if err := json.Unmarshal(data, &sponsors); err != nil {
		return nil, err
	}
	return sponsors, nil
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

//  SAVE CSV

func (a *App) SavePlayersCSV(players []map[string]string) error {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	filePath := filepath.Join(base, "frontend", "data.csv")

	// ✅ Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		f, _ := os.Create(filePath)
		defer f.Close()
		f.WriteString("name,team\n")
	}

	// Read existing data
	existing := map[string]string{}
	file, err := os.Open(filePath)
	if err == nil {
		defer file.Close()
		r := csv.NewReader(file)
		records, _ := r.ReadAll()
		for _, row := range records[1:] { // skip header
			if len(row) >= 2 {
				existing[strings.TrimSpace(row[0])] = strings.TrimSpace(row[1])
			}
		}
	}

	// Add new players
	for _, p := range players {
		name := strings.TrimSpace(p["name"])
		team := strings.TrimSpace(p["team"])
		if name == "" {
			continue
		}
		if _, exists := existing[strings.ToLower(name)]; !exists {
			existing[name] = team
		}
	}

	// Rewrite CSV
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"name", "team"})
	for name, team := range existing {
		w.Write([]string{name, team})
	}

	return nil
}

func (a *App) LoadPlayersCSV() ([]map[string]string, error) {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	filePath := filepath.Join(base, "frontend", "data.csv")

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil || len(records) < 2 {
		return []map[string]string{}, nil
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
	return players, nil
}
