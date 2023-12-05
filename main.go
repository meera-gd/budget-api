package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/hello/{username}", GreetingHandler).Methods(http.MethodGet)
	router.PathPrefix("/api").HandlerFunc(NotFoundHandler)
	router.PathPrefix("/").Handler(FrontendHandler()).Methods(http.MethodGet)

	port := os.Getenv("PORT")
	http.ListenAndServe(":"+port, router)
}

func FrontendHandler() http.Handler {
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

func GreetingHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	data := map[string]string{
		"message": "Hello, " + username + "!",
	}
	writeJSON(w, data, http.StatusOK)
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
