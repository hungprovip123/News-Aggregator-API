package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	// Serve static files from web directory
	fs := http.FileServer(http.Dir("./web/"))

	// Handle CORS for API calls
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			h.ServeHTTP(w, r)
		})
	}

	// Root handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join("web", "index.html"))
			return
		}

		fs.ServeHTTP(w, r)
	})

	port := "3000"
	fmt.Printf("🌐 Web server chạy tại: http://localhost:%s\n", port)
	fmt.Println("📱 Mở trình duyệt tại: http://localhost:3000")

	log.Fatal(http.ListenAndServe(":"+port, corsHandler(http.DefaultServeMux)))
}
