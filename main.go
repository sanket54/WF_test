package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	//"gonum.org/v1/plot"
	//"gonum.org/v1/plot/plotter"
	//"gonum.org/v1/plot/vg"
	//"github.com/gonum/stat"
	"github.com/gorilla/mux"
)

// DataPoint represents a single data point in the CSV file
type DataPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ScatterPlot represents a scatter plot
type ScatterPlot struct {
	Title      string      `json:"title"`
	XLabel     string      `json:"xlabel"`
	YLabel     string      `json:"ylabel"`
	DataPoints []DataPoint `json:"data_points"`
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// Set a limit of 5 MB for the file size
	r.ParseMultipartForm(5 * 1024 * 1024)
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to read file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if the uploaded file is a CSV file
	if handler.Header.Get("Content-Type") != "text/csv" {
		http.Error(w, "Uploaded file is not a CSV file", http.StatusBadRequest)
		return
	}

	err = os.MkdirAll("./data", os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new file in the uploads directory
	f, err := os.Create(fmt.Sprintf("./data/%s", handler.Filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer f.Close()
	// Copy the uploaded file to the new file on the server
	_, err = io.Copy(f, file)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to copy file to server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded successfully")
}

func listFiles(directory string) ([]string, error) {
	var files []string

	// Open the directory
	dir, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	// Read the directory entries
	entries, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	// Add CSV files to the list
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".csv") {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	// List CSV files in the data directory

	//fmt.Println("in list")
	files, err := listFiles("./data")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the list of files as JSON
	if err := json.NewEncoder(w).Encode(files); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func scatterHandler(w http.ResponseWriter, r *http.Request) {
	// Get the file name from the request parameters
	vars := mux.Vars(r)
	fileName := vars["fileName"]

	// Open the CSV file
	file, err := os.Open(fmt.Sprintf("./data/%s", fileName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse the CSV file
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	records = records[1:]
	//fmt.Println("print", records)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create data points from CSV data
	var dataPoints []DataPoint
	for _, record := range records {
		x, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		y, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dataPoints = append(dataPoints, DataPoint{X: x, Y: y})
	}

	// Generate scatter plot
	/*plot, err := generateScatterPlot(dataPoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save plot to PNG image
	if err := plot.Save(10*vg.Inch, 10*vg.Inch, "scatter.png"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}*/

	// Return the scatter plot data as JSON
	scatterPlot := ScatterPlot{
		Title:      "Scatter Plot",
		XLabel:     "X",
		YLabel:     "Y",
		DataPoints: dataPoints,
	}
	if err := json.NewEncoder(w).Encode(scatterPlot); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	http.ServeFile(w, r, "./ui/index.html")
}

func main() {
	// Create a new router
	router := mux.NewRouter()

	// Serve the UI static files
	//router.PathPrefix("/").Handler(http.FileServer(http.Dir("./ui/")))

	router.HandleFunc("/", IndexHandler)
	// Handle CSV file upload
	router.HandleFunc("/upload", uploadHandler).Methods("POST")
	router.HandleFunc("/list", listHandler).Methods("GET")
	router.HandleFunc("/plot/{fileName}", scatterHandler).Methods("GET")

	// Start the server
	log.Println("Listening on :8080...")
	if err := http.ListenAndServe("localhost:8080", router); err != nil {
		log.Fatal(err)
	}
}
