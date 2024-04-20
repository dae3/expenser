package main

import (
	"context"
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
	envSheetID   = "SHEET_ID"
	tZ           = "Australia/Sydney"
	pageTemplate = `
<!DOCTYPE html>
<html>
<head><title>go on, spend!</title>
<body>
<h1>enter thy expenditure of coin</h1>
</body>
</html>
	`
)

func main() {
	page := template.Must(template.New("page").Parse(pageTemplate))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := page.Execute(w, nil); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error rendering page template: %v", err)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func appendExpense(category string, description string, amount float64, ctx context.Context) (err error) {
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
					{UserEnteredValue: &sheets.ExtendedValue{StringValue: &category}},
					{UserEnteredValue: &sheets.ExtendedValue{StringValue: &description}},
					{UserEnteredValue: &sheets.ExtendedValue{NumberValue: &amount}},
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
