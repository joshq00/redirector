package main

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	apiKey          = os.Getenv("API_KEY")
	redirectPattern = os.Getenv("REDIRECT_PATTERN") // e.g. https://github.com/%s
	spreadsheetId   = os.Getenv("SPREADSHEET_ID")
	readRange       = os.Getenv("READ_RANGE") // e.g. Sheet1!A2:A
	port            = os.Getenv("PORT")
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vals, err := getValues()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		idx := rand.Intn(len(vals))
		http.Redirect(w, r, fmt.Sprintf(redirectPattern, vals[idx]), http.StatusTemporaryRedirect)
	})
}

func getValues() ([]string, error) {

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithAPIKey(apiKey))
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

func main() {
	log.Println(http.ListenAndServe(fmt.Sprintf(":%v", port), Run()))
}
