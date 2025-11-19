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
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	// Indica se o ActiveSync está habilitado para o perfil.
	ActiveSyncEnabled bool `json:"activesync_enabled"`
	// Quando presente, indica qual host de ActiveSync usar para esse usuario ( Dedicado )
	// Quando ausente, o sistema escolhe um host.
	ActiveSyncHost string `json:"activesync_host,omitempty"`
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
		var response *ProfileResponse
		switch req.Email {
		case "lucas@ghz.com.br":
			response = &ProfileResponse{
				FullName:          "Sem host definido",
				Email:             req.Email,
				ActiveSyncEnabled: true,
			}
		case "lucas-teste@ghz.com.br":
			response = &ProfileResponse{
				FullName:          "Com host definido",
				Email:             req.Email,
				ActiveSyncEnabled: true,
				ActiveSyncHost:    "cluster-a",
			}
		case "lucas-sem-acesso@ghz.com.br":
			response = &ProfileResponse{
				FullName:          "Sem acesso ActiveSync",
				Email:             req.Email,
				ActiveSyncEnabled: false,
			}
		default:
			// Resposta padrao, simulando usuario invalido
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
