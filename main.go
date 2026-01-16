package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

/* =========================
   DATABASE CONNECTIONS
========================= */

var rdsDB *sql.DB
var localDB *sql.DB

func getEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Missing environment variable: %s", key)
	}
	return val
}

func connectDB(prefix string) *sql.DB {
	dsn := "host=" + getEnv(prefix+"_HOST") +
		" port=" + getEnv(prefix+"_PORT") +
		" user=" + getEnv(prefix+"_USER") +
		" password=" + getEnv(prefix+"_PASSWORD") +
		" dbname=" + getEnv(prefix+"_NAME") +
		" sslmode=" + getEnv(prefix+"_SSLMODE")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open %s DB: %v", prefix, err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to %s DB: %v", prefix, err)
	}

	log.Printf("Connected to %s database successfully\n", prefix)
	return db
}

func initDatabases() {
	rdsDB = connectDB("RDS_DB")
	localDB = connectDB("LOCAL_DB")
	createTable(rdsDB)
	createTable(localDB)
}

func createTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS users(
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		phone TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

/* =========================
   HTTP HANDLERS
========================= */

func formHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")

	query := `
	INSERT INTO users (name, email, phone)
	VALUES ($1, $2, $3)
	`

	// Insert into RDS
	if _, err := rdsDB.Exec(query, name, email, phone); err != nil {
		http.Error(w, "Failed to store data in RDS", http.StatusInternalServerError)
		return
	}

	// Insert into Local PostgreSQL
	if _, err := localDB.Exec(query, name, email, phone); err != nil {
		http.Error(w, "Failed to store data in local DB", http.StatusInternalServerError)
		return
	}

	log.Println("Stored user data in BOTH databases:", name, email, phone)
	w.Write([]byte("User data stored in both databases successfully"))
}

/* =========================
   MAIN
========================= */

func main() {
	initDatabases()

	http.HandleFunc("/", formHandler)
	http.HandleFunc("/submit", submitHandler)

	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
