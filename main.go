package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
)

type Config struct {
	DBUsername string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
}

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func main() {
	loggingDestination := os.Getenv("LOG_DESTINATION")
	switch loggingDestination {
	case "file":
		setupFileLogging()
	case "stdout":
		setupStdoutLogging()
	default:
		log.Println("Invalid logging destination specified. Defaulting to stdout.")
		setupStdoutLogging()
	}

	log.Println("Starting application")

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	db, err := connectDB(config)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Connected to MariaDB!")

	r := mux.NewRouter()
	r.HandleFunc("/users", createUserHandler(db)).Methods("POST")
	r.HandleFunc("/users", getUsersHandler(db)).Methods("GET")
	r.HandleFunc("/", showFormHandler).Methods("GET")
	r.HandleFunc("/create", createHandler(db)).Methods("POST")
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")           // Liveness check
	r.HandleFunc("/readiness", readinessCheckHandler(db)).Methods("GET") // Readiness check

	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadConfig() (Config, error) {
	var config Config

	err := godotenv.Load(".env")
	if err == nil {
		config.DBUsername = os.Getenv("DB_USERNAME")
		config.DBPassword = os.Getenv("DB_PASSWORD")
		config.DBHost = os.Getenv("DB_HOST")
		config.DBPort = os.Getenv("DB_PORT")
		config.DBName = os.Getenv("DB_NAME")
	}

	return config, nil
}

func connectDB(config Config) (*sql.DB, error) {
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBName)
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user User
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		insertQuery := "INSERT INTO users (first_name, last_name) VALUES (?, ?)"
		_, err = db.Exec(insertQuery, user.FirstName, user.LastName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func getUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, first_name, last_name FROM users")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var user User
			err := rows.Scan(&user.ID, &user.FirstName, &user.LastName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			users = append(users, user)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

func showFormHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("form.html"))
	tmpl.Execute(w, nil)
}

func createHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")

		insertQuery := "INSERT INTO users (first_name, last_name) VALUES (?, ?)"
		_, err := db.Exec(insertQuery, firstName, lastName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func setupFileLogging() {
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error opening log file:", err)
	}
	log.SetOutput(logFile)
}

func setupStdoutLogging() {
	log.SetOutput(os.Stdout)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func readinessCheckHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "Database is not available")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}
