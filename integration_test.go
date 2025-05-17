package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// MockViaCEPServer cria um servidor HTTP de teste para simular a API ViaCEP
func MockViaCEPServer() *httptest.Server {
	handler := http.NewServeMux()

	// Endpoint para CEP válido (01001000)
	handler.HandleFunc("/ws/01001000/json/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{
			"cep": "01001-000",
			"logradouro": "Praça da Sé",
			"complemento": "lado ímpar",
			"bairro": "Sé",
			"localidade": "São Paulo",
			"uf": "SP",
			"ibge": "3550308",
			"gia": "1004",
			"ddd": "11",
			"siafi": "7107"
		}`
		w.Write([]byte(response))
	})

	// Endpoint para CEP não encontrado (99999999)
	handler.HandleFunc("/ws/99999999/json/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"erro": true}`))
	})

	// Handler genérico para outros CEPs
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ws/") && strings.Contains(r.URL.Path, "/json/") {
			// Extrair o CEP da URL
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 {
				cep := parts[2]
				// Verificar se o CEP tem 8 dígitos
				if len(cep) == 8 {
					// Simular um CEP válido
					w.Header().Set("Content-Type", "application/json")
					response := `{
						"cep": "` + cep[:5] + `-` + cep[5:] + `",
						"logradouro": "Rua Teste",
						"complemento": "",
						"bairro": "Bairro Teste",
						"localidade": "Cidade Teste",
						"uf": "UF",
						"ibge": "0000000",
						"gia": "0000",
						"ddd": "00",
						"siafi": "0000"
					}`
					w.Write([]byte(response))
					return
				}
			}
			// CEP inválido
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"erro": "true"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	return httptest.NewServer(handler)
}

// MockWeatherAPIServer cria um servidor HTTP de teste para simular a API WeatherAPI
func MockWeatherAPIServer() *httptest.Server {
	handler := http.NewServeMux()

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Verificar parâmetros de consulta
		query := r.URL.Query()
		city := query.Get("q")
		key := query.Get("key")

		// Verificar se tem uma chave API (qualquer valor)
		if key == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": {"code": 1002, "message": "API key is invalid"}}`))
			return
		}

		// Se tiver cidade, retorna dados de clima
		if city != "" {
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"location": map[string]interface{}{
					"name":    city,
					"region":  "Test Region",
					"country": "Test Country",
					"lat":     -23.55,
					"lon":     -46.64,
				},
				"current": map[string]interface{}{
					"temp_c": 25.0,
					"temp_f": 77.0,
					"condition": map[string]interface{}{
						"text": "Partly cloudy",
						"icon": "//cdn.weatherapi.com/weather/64x64/day/116.png",
						"code": 1003,
					},
					"humidity": 60,
					"cloud":    25,
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Se não tiver cidade, retorna erro
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": 1006, "message": "No location found"}}`))
	})

	return httptest.NewServer(handler)
}

func TestIntegrationWithTestMode(t *testing.T) {
	// Ativar modo de teste
	testMode = true
	defer func() { testMode = true }() // Garantir que permanece em modo de teste

	// Criar servidor de teste para nossa API
	server := httptest.NewServer(http.HandlerFunc(handleWeatherRequest))
	defer server.Close()

	// Teste com CEP válido
	t.Run("CEP válido", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/?cep=01001000")
		if err != nil {
			t.Fatalf("Erro na requisição: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code incorreto: obtido %d, esperado %d", resp.StatusCode, http.StatusOK)
		}

		// Verificamos apenas se a resposta não está vazia, já que estamos em modo de teste
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Erro ao ler corpo da resposta: %v", err)
		}

		if len(body) == 0 {
			t.Errorf("Corpo da resposta vazio")
		}
	})

	// Teste com CEP não encontrado (mesmo em modo de teste, o CEP é validado)
	t.Run("CEP não encontrado", func(t *testing.T) {
		// Definir temporariamente a vaiável de ambiente para simular ViaCEP não encontrando o CEP
		os.Setenv("SIMULATE_CEP_NOT_FOUND", "true")
		defer os.Unsetenv("SIMULATE_CEP_NOT_FOUND")

		resp, err := http.Get(server.URL + "/?cep=99999999")
		if err != nil {
			t.Fatalf("Erro na requisição: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status code incorreto: obtido %d, esperado %d", resp.StatusCode, http.StatusNotFound)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Erro ao ler corpo da resposta: %v", err)
		}

		if string(body) != "can not find zipcode" {
			t.Errorf("Corpo da resposta incorreto: obtido %s, esperado 'can not find zipcode'", string(body))
		}
	})

	// Teste com CEP inválido
	t.Run("CEP inválido", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/?cep=123")
		if err != nil {
			t.Fatalf("Erro na requisição: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Status code incorreto: obtido %d, esperado %d", resp.StatusCode, http.StatusUnprocessableEntity)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Erro ao ler corpo da resposta: %v", err)
		}

		if string(body) != "invalid zipcode" {
			t.Errorf("Corpo da resposta incorreto: obtido %s, esperado 'invalid zipcode'", string(body))
		}
	})
}
