// Simulador da API de perfil do Skymail para testes locais.
// Responde com dados de perfil fixos para qualquer email fornecido.
// Formato da requisição: {"email": "<email>"}
// Formato da resposta: {"full_name": "Bods Bodson", "email": "<email>", "activesync_enabled": true}
// NOTE: Formato precisa ser validado com a API real do Skymail.
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type ProfileRequest struct {
	Email string `json:"email"`
}

type ProfileResponse struct {
	FullName          string `json:"full_name"`
	Email             string `json:"email"`
	ActiveSyncEnabled bool   `json:"activesync_enabled"`
}

func main() {
	http.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		log.Printf("Received profile request for email: %s", req.Email)

		response := ProfileResponse{
			FullName:          "Bods Bodson",
			Email:             req.Email,
			ActiveSyncEnabled: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
