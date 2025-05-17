package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	viaCEPURL     = "https://viacep.com.br/ws/%s/json/"
	weatherAPIURL = "http://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no"
	testMode      = false // Modo de teste desativado por padrão
)

type WeatherResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type ViaCEPResponse struct {
	Cidade string `json:"localidade"`
}

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

func main() {
	// Verificar se estamos em modo de teste a partir de variável de ambiente
	testModeEnv := os.Getenv("TEST_MODE")
	if testModeEnv == "true" {
		testMode = true
		log.Println("Iniciando em modo de teste")
	}

	// Configurar rotas
	http.HandleFunc("/", handleWeatherRequest)
	http.HandleFunc("/health", handleHealthCheck)

	// Configurar porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Endpoint para verificação de saúde (health check)
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func handleWeatherRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cep := r.URL.Query().Get("cep")
	if cep == "" {
		http.Error(w, "CEP is required", http.StatusBadRequest)
		return
	}

	// Validar formato do CEP
	validCEP := regexp.MustCompile(`^\d{8}$`)
	if !validCEP.MatchString(cep) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid zipcode"))
		return
	}

	// Buscar cidade pelo CEP
	cidade, err := getCityByCEP(cep)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("can not find zipcode"))
		return
	}

	// Buscar temperatura
	tempC, err := getTemperature(cidade)
	if err != nil {
		log.Printf("Erro ao obter temperatura: %v", err)
		http.Error(w, "Error getting temperature", http.StatusInternalServerError)
		return
	}

	// Converter temperaturas
	tempF := tempC*1.8 + 32
	tempK := tempC + 273

	response := WeatherResponse{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCityByCEP(cep string) (string, error) {
	// Para testes: simular CEP não encontrado
	if os.Getenv("SIMULATE_CEP_NOT_FOUND") == "true" {
		return "", fmt.Errorf("CEP not found")
	}

	url := fmt.Sprintf(viaCEPURL, cep)

	log.Printf("Consultando CEP: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Erro ao consultar ViaCEP: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ViaCEP retornou status code: %d", resp.StatusCode)
		return "", fmt.Errorf("CEP not found")
	}

	var viaCEPResp ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		log.Printf("Erro ao decodificar resposta do ViaCEP: %v", err)
		return "", err
	}

	if viaCEPResp.Cidade == "" {
		return "", fmt.Errorf("CEP not found")
	}

	log.Printf("Cidade encontrada: %s", viaCEPResp.Cidade)
	return viaCEPResp.Cidade, nil
}

func getTemperature(cidade string) (float64, error) {
	// Modo de teste retorna valor fictício para facilitar testes
	if testMode {
		log.Printf("Usando modo de teste para cidade: %s", cidade)
		return 25.0, nil
	}

	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("WEATHER_API_KEY not set")
	}

	// Normaliza a string removendo acentos
	encodedCidade := removeAccents(cidade)

	url := fmt.Sprintf(weatherAPIURL, apiKey, encodedCidade)

	log.Printf("Consultando temperatura para %s: %s", cidade, url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Erro ao consultar API: %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API retornou status code: %d", resp.StatusCode)
		return 0, fmt.Errorf("Error getting weather data: status %d", resp.StatusCode)
	}

	var weatherResp WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		log.Printf("Erro ao decodificar resposta: %v", err)
		return 0, err
	}

	return weatherResp.Current.TempC, nil
}

// Função para remover acentos de uma string
func removeAccents(s string) string {
	replacements := map[string]string{
		"á": "a", "à": "a", "ã": "a", "â": "a", "ä": "a",
		"é": "e", "è": "e", "ê": "e", "ë": "e",
		"í": "i", "ì": "i", "î": "i", "ï": "i",
		"ó": "o", "ò": "o", "õ": "o", "ô": "o", "ö": "o",
		"ú": "u", "ù": "u", "û": "u", "ü": "u",
		"ç": "c",
		"Á": "A", "À": "A", "Ã": "A", "Â": "A", "Ä": "A",
		"É": "E", "È": "E", "Ê": "E", "Ë": "E",
		"Í": "I", "Ì": "I", "Î": "I", "Ï": "I",
		"Ó": "O", "Ò": "O", "Õ": "O", "Ô": "O", "Ö": "O",
		"Ú": "U", "Ù": "U", "Û": "U", "Ü": "U",
		"Ç": "C",
	}

	result := s
	for from, to := range replacements {
		result = strings.Replace(result, from, to, -1)
	}
	return result
}
