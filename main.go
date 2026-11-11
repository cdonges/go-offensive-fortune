package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type pageData struct {
	Fortune      template.HTML
	RequestCount int
}

func main() {
	fortuneDir := flag.String("dir", "/usr/share/games/fortunes/off/", "fortune directory")
	addr := flag.String("addr", ":6724", "listen address")
	dbPath := flag.String("db", filepath.Join("data", "requests.db"), "sqlite db path")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	db, err := openDB(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	tmpl := template.Must(template.New("index").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Offensive Fortune</title>
  <style>
    body {
      margin: 0;
      background-color: #101010;
      color: #ffffff;
      font-family: "Lucida Console", "Courier New", monospace;
      font-size: x-large;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .fortune {
      max-width: 80ch;
      padding: 2rem;
      line-height: 1.6;
      white-space: pre-wrap;
      text-align: center;
    }
  </style>
</head>
<body>
  <div class="fortune">{{.Fortune}}</div>
  <!-- Total requests logged: {{.RequestCount}} -->
</body>
</html>`))

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		fortune, err := GetRandomFortune(*fortuneDir)
		if err != nil {
			http.Error(w, "failed to read fortune: "+err.Error(), http.StatusInternalServerError)
			return
		}

		count := countRequestLogs(db)
		data := pageData{
			Fortune:      template.HTML(strings.ReplaceAll(fortune, "\n", "<br>")),
			RequestCount: count,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("render template: %v", err)
		}
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      withRequestLogging(mux, db),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func withRequestLogging(next http.Handler, db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			insertRequestLog(db, r.RemoteAddr, r.URL.Path, r.Method)
		}
		next.ServeHTTP(w, r)
	})
}
