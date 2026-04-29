package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App holds scoreboard, commentary, brackets and sponsors
type App struct {
	scoreboard Scoreboard
	commentary Commentary
	bracket    Bracket
	sponsorDir string
	ctx        context.Context
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

	Player1  string `json:"player1"`
	Team1    string `json:"team1"`
	Nation1  string `json:"nation1"`
	Score1   int    `json:"score1"`
	Visible1 bool   `json:"visible1"`

	Player2  string `json:"player2"`
	Team2    string `json:"team2"`
	Nation2  string `json:"nation2"`
	Score2   int    `json:"score2"`
	Visible2 bool   `json:"visible2"`

	Player3  string `json:"player3"`
	Team3    string `json:"team3"`
	Nation3  string `json:"nation3"`
	Score3   int    `json:"score3"`
	Visible3 bool   `json:"visible3"`

	Player4  string `json:"player4"`
	Team4    string `json:"team4"`
	Nation4  string `json:"nation4"`
	Score4   int    `json:"score4"`
	Visible4 bool   `json:"visible4"`
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

// -------------------- START.GG INTEGRATION --------------------

type StartGGTournament struct {
	Name   string         `json:"name"`
	Slug   string         `json:"slug"`
	Events []StartGGEvent `json:"events"`
}

type StartGGEvent struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	Slug     string          `json:"slug"`
	Entrants []StartGGPlayer `json:"entrants"`
}

type StartGGPlayer struct {
	Name     string `json:"name"`
	GamerTag string `json:"gamerTag"`
}

// FetchTournamentFromStartGG fetches tournament data from start.gg including all events and entrants
func (a *App) FetchTournamentFromStartGG(tournamentURL, apiKey string) (map[string]interface{}, error) {
	eventSlug, ok := extractEventSlug(tournamentURL)
	println("🎯 extractEventSlug result:", eventSlug, "ok:", ok)
	// If URL contains /event/, query that specific event directly
	if eventSlug, ok := extractEventSlug(tournamentURL); ok {
		tournamentSlug, err := extractTournamentSlug(tournamentURL)
		if err != nil {
			return nil, err
		}
		fullSlug := "tournament/" + tournamentSlug + "/event/" + eventSlug

		query := `query EventData($slug: String!) {
  event(slug: $slug) {
    id
    name
    slug
    entrants(query: {perPage: 256}) {
      nodes {
        id
        name
        participants {
          gamerTag
        }
      }
    }
    sets(perPage: 256, filters: {}) {
      nodes {
        id
        fullRoundText
        slots {
          entrant {
            name
          }
          standing {
            stats {
              score {
                value
              }
            }
          }
        }
      }
    }
  }
}`

		payload := map[string]interface{}{
			"query": query,
			"variables": map[string]string{
				"slug": fullSlug,
			},
		}

		return a.doStartGGRequest(payload, apiKey)
	}

	// Fallback: tournament-level fetch (existing behavior)
	tournamentSlug, err := extractTournamentSlug(tournamentURL)
	if err != nil {
		return nil, err
	}

	query := `query TournamentData($slug: String!) {
  tournament(slug: $slug) {
    id
    name
    slug
    events {
      id
      name
      slug
      entrants(query: {perPage: 256}) {
        nodes {
          id
          name
          participants {
            gamerTag
          }
        }
      }
    }
  }
}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]string{
			"slug": tournamentSlug,
		},
	}

	return a.doStartGGRequest(payload, apiKey)
}

func (a *App) doStartGGRequest(payload map[string]interface{}, apiKey string) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.start.gg/gql/alpha", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start.gg API error %d: %v", resp.StatusCode, result)
	}
	if errs, ok := result["errors"]; ok {
		return nil, fmt.Errorf("start.gg returned errors: %v", errs)
	}

	// TEMP DEBUG - remove later
	if raw, err := json.MarshalIndent(result, "", "  "); err == nil {
		println("🔍 START.GG RAW RESPONSE:", string(raw))
	}
	return result, nil
}

func extractTournamentSlug(url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("tournament URL is required")
	}

	// Remove protocol and www
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	url = strings.TrimPrefix(url, "start.gg/")
	url = strings.TrimSuffix(url, "/")

	// URL is now like: tournament/airdash-bash-5/event/...
	parts := strings.Split(url, "/")

	// Look for "tournament" keyword and grab the slug after it
	for i, part := range parts {
		if part == "tournament" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}

	// Fallback: if no "tournament/" prefix, treat first segment as slug
	if len(parts) > 0 && parts[0] != "" {
		return parts[0], nil
	}

	return "", fmt.Errorf("could not parse tournament slug from URL")
}

func extractEventSlug(url string) (string, bool) {
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	url = strings.TrimPrefix(url, "start.gg/")
	url = strings.TrimSuffix(url, "/")

	// URL is now like: tournament/airdash-bash-5/event/airdasher-bash-5-umvc3/...
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "event" && i+1 < len(parts) {
			return parts[i+1], true
		}
	}
	return "", false
}

// -------------------- BRACKET SETS --------------------

type BracketSet struct {
	ID      string `json:"id"`
	Round   string `json:"round"`
	Player1 string `json:"player1"`
	Team1   string `json:"team1"`
	Player2 string `json:"player2"`
	Team2   string `json:"team2"`
}

// Add this helper function
func splitNameTeam(raw string) (name, team string) {
	raw = strings.TrimSpace(raw)

	// Format 1: "TEAM | PlayerName"
	if idx := strings.Index(raw, " | "); idx != -1 {
		team = strings.TrimSpace(raw[:idx])
		name = strings.TrimSpace(raw[idx+3:])
		return
	}

	// Format 2: "TEAM PlayerName" — first word is team if it's all-caps
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) == 2 {
		first := parts[0]
		isTeam := true
		for _, ch := range first {
			if ch >= 'a' && ch <= 'z' {
				isTeam = false
				break
			}
		}
		if isTeam && len(first) <= 6 { // team tags are usually short
			team = first
			name = strings.TrimSpace(parts[1])
			return
		}
	}

	// No team found — whole thing is the name
	return raw, ""
}

// FetchEventSets fetches all sets/matches for a specific event slug
func (a *App) FetchEventSets(fullEventSlug, apiKey string) ([]BracketSet, error) {
	query := `query EventSets($slug: String!, $page: Int!) {
  event(slug: $slug) {
    name
    sets(perPage: 50, page: $page, filters: {hideEmpty: false}) {
      pageInfo {
        totalPages
      }
      nodes {
        id
        fullRoundText
        slots {
          entrant {
            name
          }
        }
      }
    }
  }
}`

	var allSets []BracketSet
	page := 1
	totalPages := 1

	for page <= totalPages {
		payload := map[string]interface{}{
			"query": query,
			"variables": map[string]interface{}{
				"slug": fullEventSlug,
				"page": page,
			},
		}

		result, err := a.doStartGGRequest(payload, apiKey)
		if err != nil {
			return nil, err
		}

		// Navigate: result["data"]["event"]["sets"]
		data, _ := result["data"].(map[string]interface{})
		event, _ := data["event"].(map[string]interface{})
		sets, _ := event["sets"].(map[string]interface{})
		pageInfo, _ := sets["pageInfo"].(map[string]interface{})

		if tp, ok := pageInfo["totalPages"].(float64); ok {
			totalPages = int(tp)
		}

		nodes, _ := sets["nodes"].([]interface{})

		println("📦 Page", page, "- nodes count:", len(nodes))
		// Print raw first node to see structure
		if len(nodes) > 0 {
			if raw, err := json.Marshal(nodes[0]); err == nil {
				println("🔍 First node raw:", string(raw))
			}
		}

		println("📦 Raw nodes count:", len(nodes)) // ADD THIS
		for _, n := range nodes {
			node, _ := n.(map[string]interface{})
			id := fmt.Sprintf("%v", node["id"])
			round, _ := node["fullRoundText"].(string)
			println("🔍 Set:", id, round) // ADD THIS

			slots, _ := node["slots"].([]interface{})

			var rawP1, rawP2 string
			if len(slots) >= 1 {
				if s, ok := slots[0].(map[string]interface{}); ok {
					if e, ok := s["entrant"].(map[string]interface{}); ok {
						if name, ok := e["name"].(string); ok {
							rawP1 = name
						}
					}
				}
			}
			if len(slots) >= 2 {
				if s, ok := slots[1].(map[string]interface{}); ok {
					if e, ok := s["entrant"].(map[string]interface{}); ok {
						if name, ok := e["name"].(string); ok {
							rawP2 = name
						}
					}
				}
			}

			p1name, p1team := splitNameTeam(rawP1)
			p2name, p2team := splitNameTeam(rawP2)

			if p1name == "" && p2name == "" {
				continue
			}

			allSets = append(allSets, BracketSet{
				ID:      id,
				Round:   round,
				Player1: p1name,
				Team1:   p1team,
				Player2: p2name,
				Team2:   p2team,
			})

		}
		page++
	}

	return allSets, nil
}

// store last fetched sets and event slug in memory so the popup window can read them
type AppState struct {
	LastSets      []BracketSet `json:"lastSets"`
	LastEventSlug string       `json:"lastEventSlug"`
	LastAPIKey    string       `json:"lastAPIKey"`
}

var appState = &AppState{}

func (a *App) GetLastSets() []BracketSet {
	return appState.LastSets
}

func (a *App) FetchAndStoreSets(fullEventSlug, apiKey string) ([]BracketSet, error) {
	println("🎯 FetchAndStoreSets called with slug:", fullEventSlug)
	sets, err := a.FetchEventSets(fullEventSlug, apiKey)
	if err != nil {
		return nil, err
	}
	appState.LastSets = sets
	appState.LastEventSlug = fullEventSlug
	appState.LastAPIKey = apiKey
	return sets, nil
}

func (a *App) OpenBracketWindow() {
	runtime.EventsEmit(a.ctx, "open-bracket-modal")
}
