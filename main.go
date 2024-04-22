package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
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
	initOIDC()
	pages := template.Must(template.New("index.html").ParseGlob("tmpl/*.html"))

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rawIDToken, err := r.Cookie("id_token")
		if err != nil || rawIDToken.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		idToken, err := verifier.Verify(r.Context(), rawIDToken.Value)
		if err != nil {
			http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
			return
		}
		var claims struct {
			Email string `json:"email"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "Failed to parse ID token claims", http.StatusInternalServerError)
			return
		}

		if err := pages.Execute(w, claims); err != nil {
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
			if os.Getenv("NO_SHEETS_API") == "" {
				if err := appendExpense(d, r.Context()); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "%v", err)
				} else {
					pages.ExecuteTemplate(w, "submit.html", d)
				}
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
