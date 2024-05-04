package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	svc     *sheets.Service
	sheetID string
)

func init() {
	if viper.GetString("no_sheets_api") != "" {
		return
	}

	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/drive")
	if err != nil {
		log.Fatalf("Unable to find Application Default Credentials: %v", err)
	}

	svc, err = sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Unable to initialize Sheets client: %v", err)
	}
	sheetID = os.Getenv(envSheetID)
	sheetID = viper.GetString("sheet_id")
	if sheetID == "" {
		log.Fatalf("%s environment variable not set", envSheetID)
	}

	_, err = getStringValuesFromNamedRange("Categories", ctx)
}

func getStringValuesFromNamedRange(rangeName string, ctx context.Context) ([]string, error) {
	var values []string

	if viper.GetString("no_sheets_api") != "" {
		values = []string{"cat1", "cat2", "cat3"}
	} else {
		req := &sheets.BatchGetValuesByDataFilterRequest{
			DataFilters: []*sheets.DataFilter{{A1Range: rangeName}},
		}
		resp, err := svc.Spreadsheets.Values.BatchGetByDataFilter(sheetID, req).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve data by named range: %v", err)
		}
		if len(resp.ValueRanges) == 0 {
			return nil, fmt.Errorf("no data found for named range: %s", rangeName)
		}

		vr := resp.ValueRanges[0].ValueRange.Values
		values = make([]string, len(vr))
		for i, va := range vr {
			v, ok := va[0].(string)
			if !ok {
				return nil, fmt.Errorf("value at index %d is not of type string", i)
			}
			values[i] = v
		}
	}
	return values, nil
}

func appendExpense(data receivedData, ctx context.Context) (err error) {
	// sheets uses spreadsheet epoch time, ie the integer parts is days since 30 December 1899
	today := time.Since(time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)).Round(time.Hour*24).Hours() / 24
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

	if viper.GetString("no_sheets_api") != "" {
		log.Println("Sheets API disabled, skipping")
	} else {
		_, err = svc.Spreadsheets.BatchUpdate(sheetID, &sheets.BatchUpdateSpreadsheetRequest{
			IncludeSpreadsheetInResponse: false,
			Requests:                     []*sheets.Request{{AppendCells: req}},
		}).Do()

		if err != nil {
			return fmt.Errorf("Batch update failed: %v", err)
		}
	}
	return nil
}
