package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

func parseStartGGLink(link string) (string, string, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", "", fmt.Errorf("start.gg event link is required")
	}
	link = strings.TrimPrefix(link, "https://")
	link = strings.TrimPrefix(link, "http://")
	link = strings.TrimPrefix(link, "www.")
	link = strings.TrimPrefix(link, "start.gg/")
	link = strings.TrimSuffix(link, "/")

	parts := strings.Split(link, "/")
	var tournamentSlug, eventSlug string
	for i, part := range parts {
		switch part {
		case "tournament":
			if i+1 < len(parts) {
				tournamentSlug = parts[i+1]
			}
		case "event":
			if i+1 < len(parts) {
				eventSlug = parts[i+1]
			}
		}
	}
	if tournamentSlug == "" && len(parts) >= 2 && parts[0] != "tournament" {
		tournamentSlug = parts[0]
		eventSlug = parts[1]
	}
	if tournamentSlug == "" || eventSlug == "" {
		return "", "", fmt.Errorf("could not parse tournament or event slug from link")
	}
	return tournamentSlug, eventSlug, nil
}

func extractScriptJSON(body []byte) ([]byte, error) {
	re := regexp.MustCompile(`(?s)<script[^>]+id=["']__NEXT_DATA__["'][^>]*>(.*?)</script>`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find embedded JSON in start.gg page")
	}
	return matches[1], nil
}

func findFirstString(data interface{}, key string) string {
	switch v := data.(type) {
	case map[string]interface{}:
		if s, ok := v[key].(string); ok {
			return s
		}
		for _, child := range v {
			if result := findFirstString(child, key); result != "" {
				return result
			}
		}
	case []interface{}:
		for _, item := range v {
			if result := findFirstString(item, key); result != "" {
				return result
			}
		}
	}
	return ""
}

func findEventId(pageObj interface{}, tournamentSlug, eventSlug string) (string, error) {
	if props, ok := pageObj.(map[string]interface{})["props"]; ok {
		if pageProps, ok := props.(map[string]interface{})["pageProps"]; ok {
			if apollo, ok := pageProps.(map[string]interface{})["__APOLLO_STATE__"]; ok {
				if apolloMap, ok := apollo.(map[string]interface{}); ok {
					for key, value := range apolloMap {
						if strings.HasPrefix(key, "Event:") {
							if eventMap, ok := value.(map[string]interface{}); ok {
								if slug, ok := eventMap["slug"].(string); ok && slug == "tournament/"+tournamentSlug+"/event/"+eventSlug {
									if id, ok := eventMap["id"].(float64); ok {
										return fmt.Sprintf("%.0f", id), nil
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("could not find event id")
}

func fetchStartGGPageEntrants(pageURL string) (map[string]interface{}, error) {
	tournamentSlug, eventSlug, err := parseStartGGLink(pageURL)
	if err != nil {
		return nil, err
	}

	// fetch event page to get eventId
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start.gg page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	jsonData, err := extractScriptJSON(body)
	if err != nil {
		return nil, err
	}

	var pageObj interface{}
	if err := json.Unmarshal(jsonData, &pageObj); err != nil {
		return nil, err
	}

	eventId, err := findEventId(pageObj, tournamentSlug, eventSlug)
	if err != nil {
		return nil, err
	}

	// build attendees URL
	attendeesURL := fmt.Sprintf("https://www.start.gg/tournament/%s/attendees?filter=%%7B%%22eventIds%%22%%3A%%5B%%22%s%%22%%5D%%7D", tournamentSlug, eventId)

	// fetch attendees page
	req2, err := http.NewRequest("GET", attendeesURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36")

	resp2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start.gg attendees page returned status %d", resp2.StatusCode)
	}

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, err
	}

	// parse HTML for names
	re := regexp.MustCompile(`<a href="/tournament/[^"]*/attendee/[^"]*">([^<]*)</a>`)
	matches := re.FindAllSubmatch(body2, -1)

	nodes := []interface{}{}
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(string(match[1]))
			nodes = append(nodes, map[string]interface{}{
				"id":           name,
				"name":         name,
				"participants": []interface{}{map[string]interface{}{"gamerTag": ""}},
			})
		}
	}

	tournamentName := findFirstString(pageObj, "name")
	eventName := strings.Replace(eventSlug, "-", " ", -1) // simple

	return map[string]interface{}{
		"data": map[string]interface{}{
			"tournament": map[string]interface{}{
				"name": tournamentName,
				"events": []map[string]interface{}{
					{
						"name":     eventName,
						"entrants": map[string]interface{}{"nodes": nodes},
					},
				},
			},
		},
	}, nil
}

func (a *App) fetchStartGGGraphQL(tournamentSlug, eventSlug, apiKey string) (map[string]interface{}, error) {
	query := `query EventEntrants($tournamentSlug:String!,$eventSlug:String!){
  tournament(slug:$tournamentSlug){
    name
    events(query:{slug:$eventSlug}) {
      id
      name
      entrants(query:{perPage:128}) {
        nodes {
          id
          name
          participants { gamerTag }
        }
      }
    }
  }
}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]string{
			"tournamentSlug": tournamentSlug,
			"eventSlug":      eventSlug,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := "https://api.start.gg/gql-public"
	if apiKey != "" {
		url = "https://api.start.gg/gql/alpha"
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
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

	return result, nil
}

func (a *App) FetchStartGGEntrants(startggLink, apiKey string) (map[string]interface{}, error) {
	if startggLink == "" {
		return nil, fmt.Errorf("start.gg event link is required")
	}

	return fetchStartGGPageEntrants(startggLink)
}
