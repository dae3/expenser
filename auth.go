package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc"
)

var (
	oidcProvider *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	state        string
	nonce        string
)

func initOIDC() {
	if os.Getenv("EXPENSER_AUTHNZ_DISABLED") != "" {
		return
	}

	var err error
	oidcProvider, err = oidc.NewProvider(context.Background(), os.Getenv("EXPENSER_OIDC_IDP_ENDPOINT"))
	if err != nil {
		log.Fatalf("Failed to get OIDC provider: %v", err)
	}
	oidcConfig := &oidc.Config{
		ClientID: os.Getenv("EXPENSER_OIDC_CLIENT_ID"),
	}
	verifier = oidcProvider.Verifier(oidcConfig)

	state, err = generateRandomString()
	if err != nil {
		log.Fatalf("Unable to generate random state: %s", err)
	}
	nonce, err = generateRandomString()
	if err != nil {
		log.Fatalf("Unable to generate random nonce: %v", err)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	authURL := fmt.Sprintf("%s?client_id=%s&response_type=id_token&scope=openid%%20email&redirect_uri=%s&state=%s&nonce=%s&response_mode=form_post", oidcProvider.Endpoint().AuthURL, os.Getenv("EXPENSER_OIDC_CLIENT_ID"), os.Getenv("EXPENSER_OIDC_CALLBACK_URL"), state, nonce)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}
	idToken := r.FormValue("id_token")
	if idToken == "" {
		http.Error(w, "ID token not found in callback", http.StatusUnauthorized)
		return
	}
	token, err := verifier.Verify(context.Background(), idToken)
	if err != nil {
		http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
		return
	}
	if token.Nonce != string(nonce) {
		http.Error(w, "Invalid nonce", http.StatusUnauthorized)
		return
	}
	if r.FormValue("state") != string(state) {
		http.Error(w, "Invalid state", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "id_token",
		Value:   idToken,
		Expires: time.Now().Add(time.Hour),
		Path:    "/",
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func generateRandomString() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		randomIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[randomIdx.Int64()]
	}
	return string(b), nil
}

func AuthorizeHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, err := authorizeRequest(r)
		if err != nil {
			if err.Error() == "no ID token found" {
				http.Redirect(w, r, "/login", http.StatusFound)
			} else {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			}
		}

		auth, err := isUserAuthorized(email)
		if err != nil || !auth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		r.Header.Add("email", email)
		h.ServeHTTP(w, r)
	})
}
func authorizeRequest(r *http.Request) (string, error) {
	if os.Getenv("EXPENSER_AUTHNZ_DISABLED") != "" {
		return "me@example.com", nil // Bypass authorization
	} else {
		rawIDToken, err := r.Cookie("id_token")
		if err != nil || rawIDToken.Value == "" {
			http.Error(w, "No ID token found", http.StatusUnauthorized)
			return "", fmt.Errorf("no ID token found") // No token, unauthorized
		}
		idToken, err := verifier.Verify(r.Context(), rawIDToken.Value)
		if err != nil {
			http.Error(w, "Failed to verify ID token", http.StatusUnauthorized)
			return "", fmt.Errorf("failed to verify ID token: %v", err) // Verification failed, unauthorized
		}
		var claims struct {
			Email string `json:"email"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "Failed to parse ID token claims", http.StatusUnauthorized)
			return "", fmt.Errorf("failed to parse ID token claims: %v", err) // Claims parsing failed, unauthorized
		}
		return claims.Email, nil // Successfully authorized
	}
}

func isUserAuthorized(email string) (bool, error) {
	if os.Getenv("EXPENSER_AUTHNZ_DISABLED") != "" {
		return true, nil
	} else {
		file, err := os.Open(os.Getenv("EXPENSER_USERFILE"))
		if err != nil {
			return false, fmt.Errorf("failed to open user file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if scanner.Text() == email {
				return true, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("error reading user file: %v", err)
		}
		return false, nil
	}
}
