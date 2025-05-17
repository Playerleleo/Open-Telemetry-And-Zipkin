package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleWeatherRequest(t *testing.T) {
	// Ativar modo de teste para todos os testes
	testMode = true

	// Tabela de testes
	tests := []struct {
		name           string
		cep            string
		expectedStatus int
		expectedBody   string
		checkJSON      bool
	}{
		{
			name:           "CEP válido",
			cep:            "01001000",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
		},
		{
			name:           "CEP inválido",
			cep:            "123",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "invalid zipcode",
		},
		{
			name:           "CEP não encontrado",
			cep:            "99999999",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "can not find zipcode",
		},
		{
			name:           "CEP vazio",
			cep:            "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	// Executar cada caso de teste
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Criar requisição de teste
			req, err := http.NewRequest("GET", "/?cep="+tt.cep, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Criar resposta de teste
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleWeatherRequest)

			// Executar requisição
			handler.ServeHTTP(rr, req)

			// Verificar status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Status code incorreto: obtido %v, esperado %v", status, tt.expectedStatus)
			}

			// Se esperamos verificar o JSON da resposta
			if tt.checkJSON {
				var response WeatherResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Falha ao decodificar JSON: %v", err)
				}

				// Verificar se temperatura em Celsius é válida
				if response.TempC <= 0 {
					t.Errorf("Temperatura Celsius inválida: %v", response.TempC)
				}

				// Verificar conversão para Fahrenheit
				expectedTempF := response.TempC*1.8 + 32
				if response.TempF != expectedTempF {
					t.Errorf("Temperatura Fahrenheit incorreta: obtido %v, esperado %v", response.TempF, expectedTempF)
				}

				// Verificar conversão para Kelvin
				expectedTempK := response.TempC + 273
				if response.TempK != expectedTempK {
					t.Errorf("Temperatura Kelvin incorreta: obtido %v, esperado %v", response.TempK, expectedTempK)
				}
			} else if tt.expectedBody != "" {
				// Verificar o corpo da resposta para erros
				if rr.Body.String() != tt.expectedBody {
					t.Errorf("Corpo da resposta incorreto: obtido %v, esperado %v",
						rr.Body.String(), tt.expectedBody)
				}
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	// Criar requisição de teste
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Criar resposta de teste
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleHealthCheck)

	// Executar requisição
	handler.ServeHTTP(rr, req)

	// Verificar status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Status code incorreto: obtido %v, esperado %v", status, http.StatusOK)
	}

	// Verificar corpo da resposta
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Falha ao decodificar JSON: %v", err)
	}

	if status, exists := response["status"]; !exists || status != "ok" {
		t.Errorf("Resposta incorreta: %v", response)
	}
}
