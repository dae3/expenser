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
		email, err := authorizeRequest(r)
		if err != nil {
			if err.Error() == "no ID token found" {
				http.Redirect(w, r, "/login", http.StatusFound)
			} else {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			}
			return
		}
		auth, err := isUserAuthorized(email)
		if err != nil || !auth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var pagedata struct {
			Email string
		}

		pagedata.Email = email

		if err := pages.Execute(w, pagedata); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error rendering page template: %v", err)
		}
	})

	http.Handle("GET /css/", http.StripPrefix("/css", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { http.ServeFileFS(w, r, os.DirFS("./css"), r.URL.Path) },
	)))

	http.HandleFunc("POST /submit", func(w http.ResponseWriter, r *http.Request) {
		email, err := authorizeRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		auth, err := isUserAuthorized(email)
		if err != nil || !auth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		log.Printf("POST /submit from %s\n", r.RemoteAddr)
		if r.Header["Content-Type"][0] != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Bad content-type")
			return
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Failed to parse form: %v', err")
			return
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
			log.Println("Valid request")
			if os.Getenv("NO_SHEETS_API") == "" {
				if err := appendExpense(d, r.Context()); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "%v", err)
					log.Printf("Sheets API error: %v\n", err)
				} else {
					log.Println("Saved to Sheets")
					pages.ExecuteTemplate(w, "submit.html", d)
				}
			} else {
				log.Println("NO_SHEETS_API set: skipping spreadsheet update")
				pages.ExecuteTemplate(w, "submit.html", d)
			}
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
