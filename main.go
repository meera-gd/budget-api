package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"firebase.google.com/go"
	"firebase.google.com/go/auth"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"google.golang.org/api/option"
)

var firebaseAuthClient *auth.Client

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	firebaseOpt := option.WithCredentialsFile(os.Getenv("FIREBASE_KEY_FILE"))
	firebaseApp, err := firebase.NewApp(context.Background(), nil, firebaseOpt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v", err)
	}

	firebaseAuthClient, err = firebaseApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Firebase auth: %v", err)
	}

	router := mux.NewRouter()
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(authMiddleware)

	apiRouter.HandleFunc("/hello/{username}", greetingHandler).Methods(http.MethodGet)
	apiRouter.PathPrefix("/").HandlerFunc(notFoundHandler)

	router.PathPrefix("/").Handler(frontendHandler()).Methods(http.MethodGet)

	port := os.Getenv("PORT")
	http.ListenAndServe(":"+port, router)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr, authHeaderHasBearerFormat := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !authHeaderHasBearerFormat || len(tokenStr) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token, err := firebaseAuthClient.VerifyIDToken(r.Context(), tokenStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "uid", token.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func frontendHandler() http.Handler {
	fileServer := os.Getenv("FILE_SERVER")
	target, err := url.Parse(fileServer)
	if err != nil {
		log.Fatal("FILE_SERVER is not a valid URL")
	}
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			// routing is handled client-side, so unless a specific file is being requested, just return index.html
			path := strings.Split(r.In.URL.Path, "/")
			filename := path[len(path)-1]
			if !strings.Contains(filename, ".") {
				r.Out.URL.Path = target.Path
			}
		},
	}
}

func greetingHandler(w http.ResponseWriter, r *http.Request) {
	//uid, ok := r.Context().Value("uid").(string)
	username := mux.Vars(r)["username"]
	data := map[string]string{
		"message": "Hello, " + username + "!",
	}
	writeJSON(w, data, http.StatusOK)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
