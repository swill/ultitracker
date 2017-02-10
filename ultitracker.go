package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/oauth2/google"
	sheets "google.golang.org/api/sheets/v4"
)

const (
	// stats cols
	NAME_COL int = 0
	TASK_COL int = 1
	TIME_COL int = 2
	DATE_COL int = 3
	NOTE_COL int = 4
)

// spreadsheet URL looks like:
// // https://docs.google.com/spreadsheets/d/1Kh7AcFON0ZGHGaeDQpqbLLIndtRrZTdD5XVTv6CTjfI/edit#gid=0
var (
	srv              *sheets.Service
	tpl              map[string]*template.Template
	use_local_static = false
	time_map         = map[string]float64{
		"15min":    0.25,
		"30min":    0.5,
		"45min":    0.75,
		"1h":       1,
		"1h 15min": 1.25,
		"1h 30min": 1.5,
		"1h 45min": 1.75,
		"2h":       2,
		"2h 15min": 2.25,
		"2h 30min": 2.5,
		"2h 45min": 2.75,
		"3h":       3,
		"3h 15min": 3.25,
		"3h 30min": 3.5,
		"3h 45min": 3.75,
		"4h":       4,
		"4h 15min": 4.25,
		"4h 30min": 4.5,
		"4h 45min": 4.75,
		"5h":       5,
		"5h 15min": 5.25,
		"5h 30min": 5.5,
		"5h 45min": 5.75,
		"6h":       6,
	}
)

func main() {
	flag.Parse() // parse flags

	// set defaults
	viper.SetDefault("player_range", "Settings!A:A")
	viper.SetDefault("task_range", "Settings!B:B")
	viper.SetDefault("stats_range", "Stats!A:E")
	viper.SetDefault("service_creds", "google-service-account.json")

	// read config from a file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	// validate we have all the config required to start
	missing_config := false
	if !viper.IsSet("team") {
		log.Println("Error: Missing required 'team' details in config file.")
		missing_config = true
	}
	if !viper.IsSet("port") {
		log.Println("Error: Missing required 'port' details in config file.")
		missing_config = true
	}
	if !viper.IsSet("spreadsheet_id") {
		log.Println("Error: Missing required 'spreadsheet_id' details in config file. EG: '1Kh7AcFON0ZGHGaeDQpqbLLIndtRrZTdD5XVTv6CTjfI'")
		missing_config = true
	}
	if missing_config {
		log.Fatal("Missing required configuration details, please update the config file.")
	}

	// set the required environment variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", viper.GetString("service_creds"))

	// setup sheet
	// details here: https://developers.google.com/identity/protocols/application-default-credentials
	client, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to setup google client: %v", err)
	}

	srv, err = sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	// get players list to validate that we have valid permissions on the specified sheet and the range works
	_, err = srv.Spreadsheets.Values.Get(viper.GetString("spreadsheet_id"), viper.GetString("player_range")).Do()
	if err != nil {
		log.Fatalf("Unable to access 'player_range': %s", err.Error())
	}

	// get tasks list to validate that we have valid permissions on the specified sheet and the range works
	_, err = srv.Spreadsheets.Values.Get(viper.GetString("spreadsheet_id"), viper.GetString("task_range")).Do()
	if err != nil {
		log.Fatalf("Unable to access 'task_range': %s", err.Error())
	}

	// get stats list to validate that we have valid permissions on the specified sheet and the range works
	_, err = srv.Spreadsheets.Values.Get(viper.GetString("spreadsheet_id"), viper.GetString("stats_range")).Do()
	if err != nil {
		log.Fatalf("Unable to access 'stats_range': %s", err.Error())
	}

	// setup template functions
	func_map := template.FuncMap{
		"fmt": func(f float64) string { return fmt.Sprintf("%.2f", f) },
		"inc": func(i int) int { return i + 1 },        // 1 based array from 0 based array
		"mod": func(i, j int) bool { return i%j == 0 }, // modulo: {{if mod $index 4}}
		"raw": func(msg interface{}) template.HTML { return template.HTML(msg.(template.HTML)) },
	}
	// setup templates
	tpl = make(map[string]*template.Template)
	tpl["index"] = template.Must(template.New("").Funcs(func_map).Parse(fmt.Sprintf("%s%s",
		FSMustString(use_local_static, "/static/views/index.html"),
		FSMustString(use_local_static, "/static/views/layout.html"))))
	tpl["leaderboard"] = template.Must(template.New("").Funcs(func_map).Parse(fmt.Sprintf("%s%s",
		FSMustString(use_local_static, "/static/views/leaderboard.html"),
		FSMustString(use_local_static, "/static/views/layout.html"))))
	tpl["error"] = template.Must(template.New("").Funcs(func_map).Parse(fmt.Sprintf("%s%s",
		FSMustString(use_local_static, "/static/views/error.html"),
		FSMustString(use_local_static, "/static/views/layout.html"))))

	// handle the url routing
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/leaderboard", handleLeaderboard)
	http.HandleFunc("/submit-entry", handleSubmitEntry)
	http.Handle("/static/", http.FileServer(FS(use_local_static)))

	// serve single root level files
	handleSingle("/favicon.ico", "/static/img/favicon.png")

	// start the web server
	log.Printf("ultitracker started - http://localhost:%d\n", viper.GetInt("port"))
	http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), nil)
}

type PageIndex struct {
	Title   string
	Date    string
	Players []string
	Tasks   []string
}

// index page (dashboard)
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		handleError(w, r, 404, "The page you are looking for does not exist.")
		return
	}

	year, month, day := time.Now().Date()
	page := &PageIndex{
		Title:   fmt.Sprintf("Add Session | %s UltiTracker", viper.GetString("team")),
		Date:    fmt.Sprintf("%d/%d/%d", day, month, year),
		Players: make([]string, 0),
		Tasks:   make([]string, 0),
	}

	// get players list
	player_resp, err := srv.Spreadsheets.Values.Get(viper.GetString("spreadsheet_id"), viper.GetString("player_range")).Do()
	if err != nil {
		log.Printf("Unable to retrieve player list from sheet. %v", err)
	}
	if len(player_resp.Values) > 0 {
		for r, row := range player_resp.Values {
			if r != 0 { // heading row
				page.Players = append(page.Players, strings.Trim(row[0].(string), " "))
			}
		}
	}

	// get tasks list
	task_resp, err := srv.Spreadsheets.Values.Get(viper.GetString("spreadsheet_id"), viper.GetString("task_range")).Do()
	if err != nil {
		log.Printf("Unable to retrieve task list from sheet. %v", err)
	}
	if len(task_resp.Values) > 0 {
		for r, row := range task_resp.Values {
			if r != 0 { // heading row
				page.Tasks = append(page.Tasks, strings.Trim(row[0].(string), " "))
			}
		}
	}

	// render the page...
	if err := tpl["index"].ExecuteTemplate(w, "layout", page); err != nil {
		log.Printf("Error executing template: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

type PageLeaderboard struct {
	Title        string
	Leaderboards LeaderboardList
}

type Leaderboard struct {
	Title   string
	Players PlayerList
	Weight  float64
}

type LeaderboardList []Leaderboard

func (l LeaderboardList) Len() int           { return len(l) }
func (l LeaderboardList) Less(i, j int) bool { return l[i].Weight < l[j].Weight }
func (l LeaderboardList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

type Player struct {
	Name  string
	Score float64
}

type PlayerList []Player

func (p PlayerList) Len() int           { return len(p) }
func (p PlayerList) Less(i, j int) bool { return p[i].Score < p[j].Score }
func (p PlayerList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PlayerList) Sum() float64 {
	total := 0.0
	for i := range p {
		total += p[i].Score
	}
	return total
}

func OrderPlayers(player_map map[string]float64) PlayerList {
	pl := make(PlayerList, len(player_map))
	i := 0
	for k, v := range player_map {
		pl[i] = Player{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

// leaderboard page
func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	page := &PageLeaderboard{
		Title:        fmt.Sprintf("Leaderboards | %s UltiTracker", viper.GetString("team")),
		Leaderboards: LeaderboardList{},
	}

	overall := make(map[string]float64)
	task_maps := make(map[string]map[string]float64)
	// get players list
	stats_resp, err := srv.Spreadsheets.Values.Get(viper.GetString("spreadsheet_id"), viper.GetString("stats_range")).Do()
	if err != nil {
		log.Printf("Unable to retrieve stats list from sheet. %s", err.Error())
	}
	if len(stats_resp.Values) > 0 {
		for r, row := range stats_resp.Values {
			if r != 0 { // heading row
				name := strings.Trim(row[NAME_COL].(string), " ")
				task := strings.Trim(row[TASK_COL].(string), " ")
				duration, err := strconv.ParseFloat(row[TIME_COL].(string), 64)
				if err != nil {
					continue
				}

				// handle overall time
				if col_val, ok := overall[name]; ok {
					overall[name] = col_val + duration
				} else {
					overall[name] = duration
				}

				// handle per task maps of player time
				if _, ok := task_maps[task]; ok {
					if player_score, ok := task_maps[task][name]; ok {
						task_maps[task][name] = player_score + duration
					} else {
						task_maps[task][name] = duration
					}
				} else {
					player := make(map[string]float64)
					player[name] = duration
					task_maps[task] = player
				}
			}
		}
	}

	// check we have an overall value and add it if we do
	if len(overall) > 0 {
		if len(overall) >= 5 {
			pl := OrderPlayers(overall)[:5]
			page.Leaderboards = append(page.Leaderboards, Leaderboard{"Overall Leaders", pl, pl.Sum()})
		} else {
			pl := OrderPlayers(overall)
			page.Leaderboards = append(page.Leaderboards, Leaderboard{"Overall Leaders", pl, pl.Sum()})
		}
	}

	// add leaderboards for the different tasks
	if len(task_maps) > 0 {
		for task, player_map := range task_maps {
			if len(player_map) > 0 {
				if len(player_map) >= 5 {
					pl := OrderPlayers(player_map)[:5]
					page.Leaderboards = append(page.Leaderboards, Leaderboard{task, pl, pl.Sum()})
				} else {
					pl := OrderPlayers(player_map)
					page.Leaderboards = append(page.Leaderboards, Leaderboard{task, pl, pl.Sum()})
				}
			}
		}
	}

	sort.Sort(sort.Reverse(page.Leaderboards))

	// render the page...
	if err := tpl["leaderboard"].ExecuteTemplate(w, "layout", page); err != nil {
		log.Printf("Error executing template: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

type SubmitEntryReq struct {
	Name     string `json:"name"`
	Task     string `json:"task"`
	Duration string `json:"duration"`
	Date     string `json:"date"`
	Notes    string `json:"notes"`
}

type SubmitEntryRes struct {
	Msg string `json:"message"`
}

// get a person's time
func handleSubmitEntry(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var req SubmitEntryReq
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// update the spreadsheet
	var vr sheets.ValueRange
	vr.Values = append(vr.Values, []interface{}{
		req.Name,
		req.Task,
		time_map[req.Duration],
		req.Date,
		req.Notes,
	})

	_, err = srv.Spreadsheets.Values.Append(viper.GetString("spreadsheet_id"), viper.GetString("stats_range"), &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		log.Printf("Failed to save the data to the spreadsheet. %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := &SubmitEntryRes{
		Msg: "The entry was successfully saved!",
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// match a url pattern and serve a single file in response.
// eg: /favicon.ico from /static/img/favicon.ico
func handleSingle(pattern string, filename string) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		b, err := FSByte(use_local_static, filename)
		if err != nil {
			log.Printf("Error serving single file: %s\n %s\n", filename, err.Error())
		}
		w.Write(b)
	})
}

type PageError struct {
	Title     string
	ErrorCode int
	ErrorDesc string
}

// handle any http error you encounter
func handleError(w http.ResponseWriter, r *http.Request, status int, desc string) {
	title := http.StatusText(status)
	if title == "" {
		title = "Unknown Error"
	}
	page := &PageError{
		Title:     title,
		ErrorCode: status,
		ErrorDesc: desc,
	}
	w.WriteHeader(page.ErrorCode)
	if err := tpl["error"].ExecuteTemplate(w, "layout", page); err != nil {
		http.Error(w, page.ErrorDesc, page.ErrorCode)
	}
}
