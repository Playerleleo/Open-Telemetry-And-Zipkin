package models

// Request representa a requisição recebida pelo Serviço A
type CEPRequest struct {
	CEP string `json:"cep"`
}

// ServiceBRequest representa a requisição enviada para o Serviço B
type ServiceBRequest struct {
	CEP string `json:"cep"`
}

// WeatherResponse representa a resposta do Serviço B com os dados de temperatura
type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}
