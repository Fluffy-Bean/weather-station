package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var Database *sql.DB

func main() {
	database, _ := sql.Open("sqlite3", "./weather.db")
	Database = database

	CreateTables()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/upload", handleUpload)

	http.ListenAndServe("localhost:8080", mux)
}

func CreateTables() {
	query := "CREATE TABLE IF NOT EXISTS weather (id INTEGER PRIMARY KEY, temperature REAL, humidity REAL, pressure REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"
	statement, _ := Database.Prepare(query)
	statement.Exec()
}

func handleRoot(writer http.ResponseWriter, request *http.Request) {
	query := "SELECT id, temperature, humidity, pressure FROM weather ORDER BY created_at DESC"
	statement, _ := Database.Prepare(query)
	rows, _ := statement.Query()

	var (
		id          int
		temperature float64
		humidity    float64
		pressure    float64
		data        []string
	)

	for rows.Next() {
		err := rows.Scan(&id, &temperature, &humidity, &pressure)
		if err != nil {
			continue
		}
		data = append(data, fmt.Sprintf("<tr><td>%v</td><td>%f</td><td>%f</td><td>%f</td></tr>", id, temperature, humidity, pressure))
	}

	html := fmt.Sprintf("<h1>Weather</h1><table><tr><th>ID</th><th>Temperature</th><th>Humidity</th><th>Pressure</th></tr>%s</table>", strings.Join(data, ""))

	io.WriteString(writer, html)
}

func handleUpload(writer http.ResponseWriter, request *http.Request) {
	temperature, err := strconv.ParseFloat(request.URL.Query().Get("temperature"), 64)
	if err != nil {
		io.WriteString(writer, "<p>Invalid temperature</p>")
		return
	}
	humidity, err := strconv.ParseFloat(request.URL.Query().Get("humidity"), 64)
	if err != nil {
		io.WriteString(writer, "<p>Invalid humidity</p>")
		return
	}
	pressure, err := strconv.ParseFloat(request.URL.Query().Get("pressure"), 64)
	if err != nil {
		io.WriteString(writer, "<p>Invalid pressure</p>")
		return
	}

	query := "INSERT INTO weather (temperature, humidity, pressure) VALUES (?, ?, ?)"
	statement, _ := Database.Prepare(query)
	_, err = statement.Exec(temperature, humidity, pressure)
	if err != nil {
		io.WriteString(writer, "<p>Failed to insert data</p>")
		return
	}

	html := fmt.Sprintf("<h1>Weather</h1><p>Temperature: %f</p><p>Humidity: %f</p><p>Pressure: %f</p>", temperature, humidity, pressure)

	io.WriteString(writer, html)
}
