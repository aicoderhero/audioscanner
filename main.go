package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Global constants
const port = ":8080"
const maxConcurrentFFProbe = 10 // Batasi hanya 4 ffprobe yang bisa berjalan bersamaan

var workerPool chan struct{} // Channel yang berfungsi sebagai semaphore/token

// --- Struktur Respon Baru ---

// ResponseSuccess digunakan untuk output berhasil
type ResponseSuccess struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// ResponseError digunakan untuk output error
type ResponseError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- Akhir Struktur Respon Baru ---

func main() {
	// 1. Inisialisasi Worker Pool
	workerPool = make(chan struct{}, maxConcurrentFFProbe)

	// 2. Definisikan Router dan Handler
	http.HandleFunc("/analyze", AnalyzeHandler)

	// 3. Mulai Server HTTP
	fmt.Printf("Web Service berjalan di http://localhost%s\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}

// AnalyzeHandler menangani permintaan HTTP ke endpoint /analyze
func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Validasi Metode
	if r.Method != http.MethodGet {
		writeJSONError(w, "Metode tidak didukung. Gunakan GET.", http.StatusMethodNotAllowed)
		return
	}

	// 2. Rate Limiting: Minta Token dari Worker Pool
	// Goroutine akan menunggu di sini jika workerPool sudah penuh
	workerPool <- struct{}{}

	// 3. Definisikan Defer untuk Mengembalikan Token (Penting!)
	// Token HARUS dikembalikan setelah pekerjaan selesai, baik berhasil atau panic.
	defer func() {
		<-workerPool // Mengembalikan token ke channel
	}()

	// 4. Ambil Query Parameter 'f'
	filePath := r.URL.Query().Get("f")
	if filePath == "" {
		writeJSONError(w, "Parameter 'f' (path file) harus disediakan.", http.StatusBadRequest)
		return
	}

	// 5. Cek Keberadaan File
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeJSONError(w, fmt.Sprintf("File tidak ditemukan di path server: %s", filePath), http.StatusNotFound)
		return
	}

	// 6. Panggil Logika Analisis Inti
	// Gunakan defer recover untuk menangkap panic dari AnalyzeAudio (termasuk error ffprobe)
	defer func() {
		if r := recover(); r != nil {
			// r adalah pesan error dari AnalyzeAudio/handleError
			message := fmt.Sprintf("%v", r)
			log.Printf("Internal Panic/Error: %s", message)
			// Status 500 karena ffprobe error dianggap error internal
			writeJSONError(w, message, http.StatusInternalServerError)
		}
	}()

	metadata := AnalyzeAudio(filePath) // Memanggil fungsi dari analyzer.go

	// 7. Kirim Respons Sukses
	writeJSONSuccess(w, metadata)
}

// writeJSONError mengirimkan respons error dalam format JSON bersih (tanpa escape \n)
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := ResponseError{
		Status:  "error",
		Message: message,
	}
	json.NewEncoder(w).Encode(resp)
}

// writeJSONSuccess mengirimkan metadata sebagai objek JSON langsung (tanpa escape \n)
func writeJSONSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := ResponseSuccess{
		Status: "success",
		Data:   data,
	}
	json.NewEncoder(w).Encode(resp)
}
