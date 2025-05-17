package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"service-b/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	viaCEPURL     = "https://viacep.com.br/ws/%s/json/"
	weatherAPIURL = "http://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no"
)

// WeatherService implementa as operações para buscar cidade por CEP e temperatura
type WeatherService struct {
	testMode bool
	client   *http.Client
	tracer   trace.Tracer
}

// NewWeatherService cria uma nova instância do serviço
func NewWeatherService() *WeatherService {
	// Verificar modo de teste
	testMode := false
	if os.Getenv("TEST_MODE") == "true" {
		testMode = true
		log.Println("Iniciando serviço em modo de teste")
	}

	return &WeatherService{
		testMode: testMode,
		client:   &http.Client{},
		tracer:   otel.GetTracerProvider().Tracer("weather-service"),
	}
}

// GetCityByCEP busca uma cidade com base no CEP
func (s *WeatherService) GetCityByCEP(ctx context.Context, cep string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "get-city-by-cep")
	defer span.End()

	span.SetAttributes(attribute.String("cep", cep))

	// Para testes: simular CEP não encontrado
	if os.Getenv("SIMULATE_CEP_NOT_FOUND") == "true" {
		return "", fmt.Errorf("CEP not found")
	}

	url := fmt.Sprintf(viaCEPURL, cep)
	log.Printf("Consultando CEP: %s", url)

	// Criar requisição com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	// Executar requisição
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Erro ao consultar ViaCEP: %v", err)
		span.RecordError(err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ViaCEP retornou status code: %d", resp.StatusCode)
		err := fmt.Errorf("CEP not found")
		span.RecordError(err)
		return "", err
	}

	var viaCEPResp models.ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		log.Printf("Erro ao decodificar resposta do ViaCEP: %v", err)
		span.RecordError(err)
		return "", err
	}

	// Checar se a resposta contém erro
	if viaCEPResp.Erro {
		err := fmt.Errorf("CEP not found")
		span.RecordError(err)
		return "", err
	}

	if viaCEPResp.Localidade == "" {
		err := fmt.Errorf("City not found")
		span.RecordError(err)
		return "", err
	}

	log.Printf("Cidade encontrada: %s", viaCEPResp.Localidade)
	span.SetAttributes(attribute.String("city", viaCEPResp.Localidade))
	return viaCEPResp.Localidade, nil
}

// GetTemperature busca a temperatura para uma cidade
func (s *WeatherService) GetTemperature(ctx context.Context, cidade string) (float64, error) {
	ctx, span := s.tracer.Start(ctx, "get-temperature")
	defer span.End()

	span.SetAttributes(attribute.String("city", cidade))

	// Modo de teste retorna valor fictício para facilitar testes
	if s.testMode {
		log.Printf("Usando modo de teste para cidade: %s", cidade)
		return 25.0, nil
	}

	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		err := fmt.Errorf("WEATHER_API_KEY not set")
		span.RecordError(err)
		return 0, err
	}

	// Normaliza a string removendo acentos
	encodedCidade := s.removeAccents(cidade)

	url := fmt.Sprintf(weatherAPIURL, apiKey, encodedCidade)
	log.Printf("Consultando temperatura para %s: %s", cidade, url)

	// Criar requisição com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	// Executar requisição
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Erro ao consultar API: %v", err)
		span.RecordError(err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API retornou status code: %d", resp.StatusCode)
		err := fmt.Errorf("Error getting weather data: status %d", resp.StatusCode)
		span.RecordError(err)
		return 0, err
	}

	var weatherResp models.WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		log.Printf("Erro ao decodificar resposta: %v", err)
		span.RecordError(err)
		return 0, err
	}

	// Registrar a temperatura encontrada
	span.SetAttributes(attribute.Float64("temperature_c", weatherResp.Current.TempC))
	return weatherResp.Current.TempC, nil
}

// Função para remover acentos de uma string
func (s *WeatherService) removeAccents(texto string) string {
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

	result := texto
	for from, to := range replacements {
		result = strings.Replace(result, from, to, -1)
	}
	return result
}
