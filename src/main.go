package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	envSheetID         = "SHEET_ID"
	tZ                 = "Australia/Sydney"
	formFieldMaxLength = 256
)

type receivedData struct {
	Category    string
	Description string
	Amount      float64
}

// go:embed
var templates embed.FS

func truncatedFormStringValue(r *http.Request, fieldName string) (error, string) {
	val := r.Form[fieldName]
	if val == nil || val[0] == "" {
		return errors.New(fmt.Sprintf("Field %s not present in form", fieldName)), ""
	}

	if len(val[0]) > formFieldMaxLength {
		return nil, string([]rune(val[0])[:formFieldMaxLength])
	}
	return nil, val[0]
}

func main() {
	pages := template.Must(template.New("index.html").ParseGlob("tmpl/*.html"))
	http.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
		if err := pages.Execute(w, nil); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error rendering page template: %v", err)
		}
	})

	http.Handle("GET /css/", http.StripPrefix("/css", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { http.ServeFileFS(w, r, os.DirFS("./css"), r.URL.Path) },
	)))

	http.HandleFunc("POST /submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Bad content-type")
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to parse form: %v', err")
		}

		var d receivedData
		var errc, errd error
		// some dumb input protections
		errc, d.Category = truncatedFormStringValue(r, "category")
		errd, d.Description = truncatedFormStringValue(r, "description")
		erra, amountStr := truncatedFormStringValue(r, "amount")
		n, err := fmt.Sscanf(amountStr, "%f", &d.Amount)
		if errc != nil || errd != nil || erra != nil || n == 0 || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "errc: %v\nerrd: %v\nerra: %v\nn: %d\nerr: %v", errc, errd, erra, n, err)
		} else {
			if err := appendExpense(d, r.Context()); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "%v", err)
			} else {
				pages.ExecuteTemplate(w, "submit.html", d)
			}
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func appendExpense(data receivedData, ctx context.Context) (err error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/drive")
	if err != nil {
		return fmt.Errorf("Unable to find Application Default Credentials: %v", err)
	}

	svc, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("Unable to initialize Sheets client: %v", err)
	}
	sheetID := os.Getenv(envSheetID)
	if sheetID == "" {
		return fmt.Errorf("%s environment variable not set", envSheetID)
	}

	// sheets uses spreadsheet epoch time, ie the integer parts is days since 30 December 1899
	today := time.Since(time.Date(1899, 12, 30, 0, 0, 0, 0, time.FixedZone(tZ, 0))).Hours() / 24
	emptyString := ""

	req := &sheets.AppendCellsRequest{
		Fields:  "*",
		SheetId: 1,
		Rows: []*sheets.RowData{
			{
				ForceSendFields: nil,
				NullFields:      nil,
				Values: []*sheets.CellData{
					{
						UserEnteredValue:  &sheets.ExtendedValue{NumberValue: &today},
						UserEnteredFormat: &sheets.CellFormat{NumberFormat: &sheets.NumberFormat{Type: "DATE"}},
					},
					{UserEnteredValue: &sheets.ExtendedValue{StringValue: &emptyString}},
					{UserEnteredValue: &sheets.ExtendedValue{StringValue: &data.Category}},
					{UserEnteredValue: &sheets.ExtendedValue{StringValue: &data.Description}},
					{UserEnteredValue: &sheets.ExtendedValue{NumberValue: &data.Amount}},
				},
			},
		},
	}
	_, err = svc.Spreadsheets.BatchUpdate(sheetID, &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: false,
		Requests:                     []*sheets.Request{{AppendCells: req}},
	}).Do()

	if err != nil {
		return fmt.Errorf("Batch update failed: %v", err)
	}

	return nil
}