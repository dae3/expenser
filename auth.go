package main

import (
	"context"
	"fmt"
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
	authURL := fmt.Sprintf("%s?client_id=%s&response_type=id_token&scope=openid%20email&redirect_uri=http://localhost:8080/callback&state=%s&nonce=%s&response_mode=query", oidcProvider.Endpoint().AuthURL, os.Getenv("OIDC_CLIENT_ID"), state, nonce)
	http.Redirect(w, r, authURL, http.StatusFound)
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
