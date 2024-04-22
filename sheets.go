package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

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
