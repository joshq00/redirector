package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/sheets/v4"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	apiKey          = strings.TrimSpace(os.Getenv("API_KEY"))
	redirectPattern = strings.TrimSpace(os.Getenv("REDIRECT_PATTERN")) // e.g. https://github.com/%s
	spreadsheetId   = strings.TrimSpace(os.Getenv("SPREADSHEET_ID"))
	readRange       = strings.TrimSpace(os.Getenv("READ_RANGE")) // e.g. Sheet1!A2:A
	port            = strings.TrimSpace(os.Getenv("PORT"))
	host            = strings.TrimSpace(os.Getenv("HOST"))
)

func init() {
	if apiKey == "" {
		panic("API_KEY must be set")
	}
	if spreadsheetId == "" {
		panic("SPREADSHEET_ID must be set")
	}
	if redirectPattern == "" {
		panic("REDIRECT_PATTERN must be set")
	}
	if readRange == "" {
		panic("READ_RANGE must be set")
	}
	if port == "" {
		panic("PORT must be set")
	}
	rand.Seed(time.Now().UnixNano())
}

func Run() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		vals, err := getValues()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		idx := rand.Intn(len(vals))
		http.Redirect(w, r, determineRedirectURI(redirectPattern, vals[idx]), http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/uris", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listAll(w, r)
		case http.MethodPost:
			addUri(w, r)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	return mux
}

func addUri(w http.ResponseWriter, r *http.Request) {
	bod := struct {
		Value string `json:"value"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&bod); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := addValues([]string{bod.Value}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(toLink(bod.Value))
}

type link struct {
	Value string `json:"value"`
	URI   string `json:"uri"`
}

func toLink(v string) link {
	return link{
		v,
		determineRedirectURI(redirectPattern, v),
	}
}

func listAll(w http.ResponseWriter, r *http.Request) {
	vals, err := getValues()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data := make([]link, len(vals))
	for i, v := range vals {
		data[i] = toLink(v)
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(struct {
		Data []link `json:"data"`
	}{data})
}

func determineRedirectURI(pat, val string) string {
	uri, err := url.Parse(val)
	if err == nil && uri.Scheme != "" && uri.Host != "" {
		return uri.String()
	}
	return fmt.Sprintf(pat, url.PathEscape(val))
}

func getValues() ([]string, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return nil, err
	}

	vals := []string{}
	for _, row := range resp.Values {
		for _, val := range row {
			vals = append(vals, fmt.Sprintf("%s", val))
		}
	}
	return vals, nil
}

func addValues(vals []string) error {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	values := &sheets.ValueRange{
		Values: [][]interface{}{},
	}
	for _, v := range vals {
		values.Values = append(values.Values, []interface{}{v})
	}
	_, err = srv.Spreadsheets.Values.
		Append(spreadsheetId, readRange, values).
		ValueInputOption("RAW").
		Do()
	return err
}

func main() {
	addr := fmt.Sprintf("%v:%v", host, port)
	log.Println("listening on", addr)
	log.Println(http.ListenAndServe(addr, Run()))
}
