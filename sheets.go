package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	svc     *sheets.Service
	sheetID string
)

func init() {
	if os.Getenv("EXPENSER_NO_SHEETS_API") != "" {
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
	if sheetID == "" {
		log.Fatalf("%s environment variable not set", envSheetID)
	}

	vr, err := getNamedRange("Categories", ctx)
	if err != nil {
		log.Fatalf("Unable to get named range: %v", err)
	}
	_, err = getStringValuesFromRange(vr)
	if err != nil {
		log.Fatalf("Unable to get string values from range: %v", err)
	}
}

func getNamedRange(rangeName string, ctx context.Context) (*sheets.ValueRange, error) {
	if os.Getenv("EXPENSER_NO_SHEETS_API") == "" {
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

		return resp.ValueRanges[0].ValueRange, nil
	} else {
		dummy := &sheets.ValueRange{}
		dummy.Values = make([][]interface{}, 1)
		dummy.Values[0] = make([]interface{}, 1)
		dummy.Values[0][0] = "dummy"
		return dummy, nil
	}
}

func getStringValuesFromRange(vr *sheets.ValueRange) ([]string, error) {
	values := make([]string, len(vr.Values))
	for i, va := range vr.Values {
		v, ok := va[0].(string)
		if !ok {
			return nil, fmt.Errorf("value at index %d is not of type string", i)
		}
		values[i] = v
	}
	return values, nil
}

func getArrayFromRange(vr *sheets.ValueRange, columns int) ([][]string, error) {
	array := make([][]string, len(vr.Values))
	for i, va := range vr.Values {
		array[i] = make([]string, columns)
		for j, aa := range va {
			v, ok := aa.(string)
			if !ok {
				return nil, fmt.Errorf("value at index (%d, %d) is not of type string", i)
			}
			array[i][j] = v
		}
	}
	return array, nil
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

	if os.Getenv("EXPENSER_NO_SHEETS_API") != "" {
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
