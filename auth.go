package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc"
)

var (
	oidcProvider *oidc.Provider
	verifier     *oidc.IDTokenVerifier
)

func initOIDC() {
	var err error
	oidcProvider, err = oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		log.Fatalf("Failed to get OIDC provider: %v", err)
	}
	oidcConfig := &oidc.Config{
		ClientID: os.Getenv("OIDC_CLIENT_ID"),
	}
	verifier = oidcProvider.Verifier(oidcConfig)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	state := "example-state" // This should be a random or session-specific value in production
	nonce := "example-nonce" // This should also be a random or session-specific value
	http.Redirect(w, r, oidcProvider.Endpoint().AuthURL+"?client_id="+os.Getenv("OIDC_CLIENT_ID")+"&response_type=id_token&scope=openid email&redirect_uri=http://localhost:8080/callback&state="+state+"&nonce="+nonce, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	idToken := r.URL.Query().Get("id_token")
	if idToken == "" {
		http.Error(w, "ID token not found in callback", http.StatusUnauthorized)
		return
	}
	_, err := verifier.Verify(context.Background(), idToken)
	if err != nil {
		http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "id_token",
		Value:   idToken,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc"
)

var (
	oidcProvider *oidc.Provider
	verifier     *oidc.IDTokenVerifier
)

func initOIDC() {
	var err error
	oidcProvider, err = oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		log.Fatalf("Failed to get OIDC provider: %v", err)
	}
	oidcConfig := &oidc.Config{
		ClientID: os.Getenv("OIDC_CLIENT_ID"),
	}
	verifier = oidcProvider.Verifier(oidcConfig)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	state := "example-state" // This should be a random or session-specific value in production
	nonce := "example-nonce" // This should also be a random or session-specific value
	http.Redirect(w, r, oidcProvider.Endpoint().AuthURL+"?client_id="+os.Getenv("OIDC_CLIENT_ID")+"&response_type=id_token&scope=openid email&redirect_uri=http://localhost:8080/callback&state="+state+"&nonce="+nonce, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	idToken := r.URL.Query().Get("id_token")
	if idToken == "" {
		http.Error(w, "ID token not found in callback", http.StatusUnauthorized)
		return
	}
	_, err := verifier.Verify(context.Background(), idToken)
	if err != nil {
		http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "id_token",
		Value:   idToken,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
