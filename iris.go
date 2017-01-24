package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	sheets "google.golang.org/api/sheets/v4"

	"golang.org/x/oauth2/google"

	"github.com/rakyll/globalconf"
)

// spreadsheet URL looks like:
// // https://docs.google.com/spreadsheets/d/1Kh7AcFON0ZGHGaeDQpqbLLIndtRrZTdD5XVTv6CTjfI/edit#gid=0
var (
	conf             *globalconf.GlobalConf
	srv              *sheets.Service
	tpl              map[string]*template.Template
	use_local_static = false
	iris_path        = ""
	port             = flag.Int("port", 8000, "The port the timework app should listen on")
	debug            = flag.Bool("debug", false, "Print logging to STDOUT for debugging")
	spreadsheet_id   = flag.String("spreadsheet_id", "1Kh7AcFON0ZGHGaeDQpqbLLIndtRrZTdD5XVTv6CTjfI", "The Spreadsheet ID")
	player_range     = flag.String("player_range", "Settings!A:A", "The player column identifier")
	task_range       = flag.String("task_range", "Settings!B:B", "The task column identifier")
	stats_range      = flag.String("stats_range", "Stats!A:E", "The columns to store the stats")
	time_map         = map[string]float64{
		"10min":    0.17,
		"20min":    0.33,
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
	var err error

	// setup a project directory in the HOME direcotry for logs, etc...
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Could not get current user.\nError:%s", err.Error())
		log.Println("Could not get current user.\nError:%s", err.Error())
		os.Exit(1)
	}
	iris_path = filepath.Join(usr.HomeDir, ".iris")
	err = os.MkdirAll(iris_path, 0777)
	if err != nil {
		fmt.Println("Could not create directory:%s\nError:%s", iris_path, err.Error())
		log.Println("Could not create directory:%s\nError:%s", iris_path, err.Error())
		os.Exit(1)
	}

	if !*debug {
		// setup the application log in the 'HOME/.iris' directory
		log_path := filepath.Join(iris_path, "iris.log")
		log_file, err := os.OpenFile(log_path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Error creating log: '%s' - %s\n", log_path, err.Error())
			log.Printf("Error creating log: '%s' - %s\n", log_path, err.Error())
			os.Exit(1)
		}
		defer log_file.Close()
		log.SetOutput(log_file)
	}

	// setup the configuration file in the 'HOME/.iris' directory
	conf_path := filepath.Join(iris_path, "iris.conf")
	conf_file, err := os.OpenFile(conf_path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666) // create if needed
	if err != nil {
		fmt.Printf("Error creating config file: '%s' - %s\n", err.Error())
		log.Printf("Error creating config file: '%s' - %s\n", err.Error())
		os.Exit(1)
	}
	conf_file.Close() // close right away and give control to globalconf now that we have a file for sure
	conf, err = globalconf.NewWithOptions(&globalconf.Options{
		Filename: conf_path,
	})
	conf.ParseAll()

	// setup sheet
	ctx := context.Background()
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to setup google client: %v", err)
	}

	srv, err = sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	// setup template functions
	func_map := template.FuncMap{
		"raw": func(msg interface{}) template.HTML { return template.HTML(msg.(template.HTML)) },
	}
	// setup templates
	tpl = make(map[string]*template.Template)
	tpl["index"] = template.Must(template.New("").Funcs(func_map).Parse(fmt.Sprintf("%s%s",
		FSMustString(use_local_static, "/static/views/index.html"),
		FSMustString(use_local_static, "/static/views/layout.html"))))
	tpl["error"] = template.Must(template.New("").Funcs(func_map).Parse(fmt.Sprintf("%s%s",
		FSMustString(use_local_static, "/static/views/error.html"),
		FSMustString(use_local_static, "/static/views/layout.html"))))

	// handle the url routing
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/submit-entry", handleSubmitEntry)
	http.Handle("/static/", http.FileServer(FS(use_local_static)))

	// serve single root level files
	handleSingle("/favicon.ico", "/static/img/iris_logo_inverse.png")

	fmt.Printf("URL: http://localhost:%d\n", *port)
	fmt.Println("server started...")

	// start the web server
	log.Printf("iris started - localhost:%d\n", *port)
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
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
		Title:   "Enter Event | Iris",
		Date:    fmt.Sprintf("%d/%d/%d", day, month, year),
		Players: make([]string, 0),
		Tasks:   make([]string, 0),
	}

	// get players list
	player_resp, err := srv.Spreadsheets.Values.Get(*spreadsheet_id, *player_range).Do()
	if err != nil {
		log.Printf("Unable to retrieve player list from sheet. %v", err)
	}
	if len(player_resp.Values) > 0 {
		for r, row := range player_resp.Values {
			if r != 0 {
				page.Players = append(page.Players, strings.Trim(row[0].(string), " "))
			}
		}
	}

	// get tasks list
	task_resp, err := srv.Spreadsheets.Values.Get(*spreadsheet_id, *task_range).Do()
	if err != nil {
		log.Printf("Unable to retrieve task list from sheet. %v", err)
	}
	if len(task_resp.Values) > 0 {
		for r, row := range task_resp.Values {
			if r != 0 {
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

	_, err = srv.Spreadsheets.Values.Append(*spreadsheet_id, *stats_range, &vr).ValueInputOption("USER_ENTERED").Do()
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
