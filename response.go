package main

// Response (Struktur lama yang kini hanya sebagai placeholder,
// namun dipertahankan untuk modularitas dan menghindari error kompilasi).
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
