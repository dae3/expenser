package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
)

const (
	envSheetID              = "EXPENSER_SHEET_ID"
	formFieldMaxLength      = 256
	favouriteQueryParameter = "fav"
)

var (
	pages *template.Template
)

var (
	pages *template.Template
)

type receivedData struct {
	Category    string
	Description string
	Amount      float64
}

func toFloat(s string) float64 {
	s = strings.TrimSpace(s)
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Error converting string '%s' to float: %v", s, err)
		return 0
	}
	return value
}

func truncatedFormStringValue(r *http.Request, fieldName string, mandatory bool) (error, string) {
	val := r.Form[fieldName]
	if val == nil || val[0] == "" {
		if mandatory {
			return errors.New(fmt.Sprintf("Field %s not present in form", fieldName)), ""
		} else {
			return nil, ""
		}
	}

	if len(val[0]) > formFieldMaxLength {
		return nil, string([]rune(val[0])[:formFieldMaxLength])
	}
	return nil, val[0]
}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	var pagedata struct {
		Categories []string
		Favourites [][]string
		Email      string
	}

	pagedata.Email = r.Header.Get("email")
	catrange, err := getNamedRange("Categories", r.Context())
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error getting category list: %v", err)
		return
	}
	cat, err := getStringValuesFromRange(catrange)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error getting category list: %v", err)
		return
	}
	pagedata.Categories = cat

	favrange, err := getNamedRange("Favourites", r.Context())
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error getting favourite list: %v", err)
		return
	}

	favs, err := getArrayFromRange(favrange, 3)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error getting category list: %v", err)
		return
	}

	pagedata.Favourites = favs

	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if q.Has(favouriteQueryParameter) {
		selectedFav, err := strconv.ParseInt(q.Get(favouriteQueryParameter), 10, 0)
		if err != nil || int(selectedFav) > len(cat) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Bad favourite parameter")
			return
		}
		pagedata.FavouriteCategory = favs[selectedFav][0]
		pagedata.FavouriteDescription = favs[selectedFav][1]
		pagedata.FavouriteAmount = favs[selectedFav][2]
	}

	if err := pages.Execute(w, pagedata); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error rendering page template: %v", err)
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
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
	errc, d.Category = truncatedFormStringValue(r, "category", true)
	errd, d.Description = truncatedFormStringValue(r, "description", false)
	erra, amountStr := truncatedFormStringValue(r, "amount", true)
	n, err := fmt.Sscanf(amountStr, "%f", &d.Amount)
	if errc != nil || errd != nil || erra != nil || n == 0 || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "errc: %v\nerrd: %v\nerra: %v\nn: %d\nerr: %v", errc, errd, erra, n, err)
	} else {
		log.Println("Valid request")
		if err := appendExpense(d, r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
			log.Printf("Sheets API error: %v\n", err)
		} else {
			log.Println("Saved to Sheets")
			pages.ExecuteTemplate(w, "submit.html", d)
		}
	}
}

func main() {
	initOIDC()
	pages = template.Must(template.New("index.html").Funcs(template.FuncMap{"toFloat": toFloat}).ParseGlob("tmpl/*.html"))

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)

	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "tmpl/manifest.json.tmpl")
	})

	http.HandleFunc("POST /api/train", TrainApiHandler)
	http.Handle("/", AuthorizeHandler(http.HandlerFunc(rootHandler)))
	http.Handle("POST /submit", AuthorizeHandler(http.HandlerFunc(submitHandler)))

	http.Handle("GET /static/", http.StripPrefix("/static", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { http.ServeFileFS(w, r, os.DirFS("./static"), r.URL.Path) },
	)))

	// service worker has to be served from / to have access to that scope
	http.HandleFunc("GET /worker.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/worker.js")
	})

	port := os.Getenv("EXPENSER_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
