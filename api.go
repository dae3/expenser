package main

import (
	"log"
	"net/http"
	"os"
)

func TrainApiHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("EXPENSER_API_KEY")
	reqApiKey := r.Header.Get("X-API-Key")

	log.Println("Train API request")

	if apiKey == "" {
		http.Error(w, "API not configured", http.StatusServiceUnavailable)
		log.Println("Train API not configured")
		return
	}
	if reqApiKey != apiKey {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		log.Println("Invalid API key")
		return
	}

	expense := receivedData{
		Category:    "Car",
		Description: "Transport/tolls/parking",
		Amount:      16.60,
	}
	err := appendExpense(expense, r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Sheets API call failed: %v\n", err)
	} else {
		w.WriteHeader(http.StatusNoContent)
		log.Println("Submitted")
	}
}
