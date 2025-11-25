package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
)

// StreamDetail adalah struktur untuk menangkap detail stream audio.
type StreamDetail struct {
	CodecName     string            `json:"codec_name"`
	CodecType     string            `json:"codec_type"`
	SampleRate    string            `json:"sample_rate"`
	BitRate       string            `json:"bit_rate"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	Tags          map[string]string `json:"tags"`
}

// FFProbeResult adalah struktur untuk menangkap output ffprobe JSON.
type FFProbeResult struct {
	Format struct {
		Filename   string            `json:"filename"`
		FormatName string            `json:"format_name"`
		Size       string            `json:"size"`
		Duration   string            `json:"duration"`
		ProbeScore int               `json:"probe_score"`
		Tags       map[string]string `json:"tags"`
	} `json:"format"`
	Streams []StreamDetail `json:"streams"`
}

// AudioMetadata adalah struktur data hasil akhir yang sudah bersih, mencakup semua metadata.
type AudioMetadata struct {
	// --- Metadata Level Stream (Teknis) ---
	FileName        string  `json:"file_name"`
	ContainerFormat string  `json:"container_format"`
	FileSizeMB      float64 `json:"file_size_mb"`
	DurationSec     float64 `json:"duration_sec"`
	IntegritasScore int     `json:"integritas_score_100"`
	Codec           string  `json:"codec"`
	SampleRateHz    int     `json:"sample_rate_hz"`
	BitRateKbps     int     `json:"bit_rate_kbps"`
	ChannelLayout   string  `json:"channel_layout"`

	// --- Metadata Level Tag (ID3/Non-Teknis) ---
	Title     string `json:"title,omitempty"`
	Artist    string `json:"artist,omitempty"`
	Album     string `json:"album,omitempty"`
	Genre     string `json:"genre,omitempty"`
	Year      string `json:"year,omitempty"`
	Track     string `json:"track,omitempty"`
	Composer  string `json:"composer,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Copyright string `json:"copyright,omitempty"`
}

// handleError di analyzer.go sekarang menggunakan panic
// agar errornya ditangkap oleh recover() di AnalyzeHandler di main.go.
func handleError(message string) {
	panic(message)
}

// AnalyzeAudio menjalankan ffprobe dan memproses hasilnya.
func AnalyzeAudio(filePath string) AudioMetadata {
	// 1. Cek Ketersediaan ffprobe
	if _, err := exec.LookPath("ffprobe"); err != nil {
		handleError("Perintah ffprobe tidak ditemukan. Pastikan FFmpeg terinstal di sistem PATH.")
	}

	// 2. Jalankan ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		handleError(fmt.Sprintf("Gagal menjalankan ffprobe: %s", err))
	}

	// 3. Urai Output ffprobe JSON
	var result FFProbeResult
	if err := json.Unmarshal(output, &result); err != nil {
		handleError("Gagal mengurai output JSON dari ffprobe: " + err.Error())
	}

	// 4. Temukan Stream Audio
	var audioStream *StreamDetail
	for i := range result.Streams {
		if result.Streams[i].CodecType == "audio" {
			audioStream = &result.Streams[i]
			break
		}
	}

	if audioStream == nil {
		handleError("Tidak ditemukan stream audio dalam file.")
	}

	// 5. Panggil parseMetadata dengan ARGUMEN yang benar
	return parseMetadata(&result, audioStream)
}

// parseMetadata melakukan konversi dari string ffprobe ke tipe numerik dan merangkum semua tag.
func parseMetadata(probeResult *FFProbeResult, stream *StreamDetail) AudioMetadata {

	// --- Parsing Data Teknis (Stream Level) ---
	durationSec, _ := strconv.ParseFloat(probeResult.Format.Duration, 64)
	sizeBytes, _ := strconv.ParseFloat(probeResult.Format.Size, 64)
	sizeMB := sizeBytes / (1024 * 1024)

	sampleRateHz, _ := strconv.Atoi(stream.SampleRate)
	bitRate, _ := strconv.Atoi(stream.BitRate)
	bitRateKbps := bitRate / 1000

	metadata := AudioMetadata{
		// Data Teknis
		FileName:        filepath.Base(probeResult.Format.Filename),
		ContainerFormat: probeResult.Format.FormatName,
		FileSizeMB:      sizeMB,
		DurationSec:     durationSec,
		IntegritasScore: probeResult.Format.ProbeScore,
		Codec:           stream.CodecName,
		SampleRateHz:    sampleRateHz,
		BitRateKbps:     bitRateKbps,
		ChannelLayout:   stream.ChannelLayout,
	}

	// --- Mengambil Semua Tag (ID3) yang Mungkin Ada ---
	if stream.Tags != nil {
		metadata.Title = stream.Tags["title"]
		metadata.Artist = stream.Tags["artist"]
		metadata.Album = stream.Tags["album"]
		metadata.Genre = stream.Tags["genre"]
		metadata.Year = stream.Tags["year"]
		metadata.Track = stream.Tags["track"]
		metadata.Composer = stream.Tags["composer"]
		metadata.Comment = stream.Tags["comment"]
		metadata.Copyright = stream.Tags["copyright"]
	}

	// Cek Tag Level Format (Fallback)
	if probeResult.Format.Tags != nil && metadata.Title == "" {
		metadata.Title = probeResult.Format.Tags["title"]
		metadata.Artist = probeResult.Format.Tags["artist"]
		metadata.Album = probeResult.Format.Tags["album"]
	}

	return metadata
}
