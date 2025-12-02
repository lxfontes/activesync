// Simulador da API de perfil do Skymail para testes locais.
// Responde com dados de perfil fixos para qualquer email fornecido.
// Formato da requisição: /emailUser/protocols/{email}
// Formato da resposta: {"result":{"success":1,"protocols":{"activesync":true}}}
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Profile struct {
	Success   int             `json:"success"`
	Protocols map[string]bool `json:"protocols"`
}

type ProfileResponse struct {
	Result Profile `json:"result"`
}

func main() {
	http.HandleFunc("/emailUser/protocols/{email}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		reqEmail := r.PathValue("email")
		if reqEmail == "" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		log.Printf("Received profile request for email: %s", reqEmail)
		var response *ProfileResponse
		switch reqEmail {
		case "lucas@ghz.com.br":
			response = &ProfileResponse{
				Result: Profile{
					Success: 1,
					Protocols: map[string]bool{
						"active_sync": true,
					},
				},
			}
		// case "lucas-teste@ghz.com.br":
		// 	response = &ProfileResponse{
		// 		FullName:          "Com host definido",
		// 		Email:             reqEmail,
		// 		ActiveSyncEnabled: true,
		// 		ActiveSyncHost:    "cluster-a",
		// 	}
		case "lucas-sem-acesso@ghz.com.br":
			response = &ProfileResponse{
				Result: Profile{
					Success: 1,
					Protocols: map[string]bool{
						"active_sync": false,
					},
				},
			}
		default:
			// Resposta padrao, simulando usuario invalido
			response = &ProfileResponse{
				Result: Profile{
					Success: 1,
					Protocols: map[string]bool{
						"active_sync": true,
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
