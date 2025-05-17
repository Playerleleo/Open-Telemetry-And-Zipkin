package models

// Requisição recebida do Serviço A
type CEPRequest struct {
	CEP string `json:"cep"`
}

// Resposta da consulta ViaCEP
type ViaCEPResponse struct {
	Cep         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"` // Nome da cidade
	Uf          string `json:"uf"`
	Ibge        string `json:"ibge"`
	Gia         string `json:"gia"`
	Ddd         string `json:"ddd"`
	Siafi       string `json:"siafi"`
	Erro        bool   `json:"erro,omitempty"`
}

// Resposta da WeatherAPI
type WeatherAPIResponse struct {
	Location struct {
		Name    string  `json:"name"`
		Region  string  `json:"region"`
		Country string  `json:"country"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
			Code int    `json:"code"`
		} `json:"condition"`
	} `json:"current"`
}

// Resposta final com os dados de temperatura
type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}
