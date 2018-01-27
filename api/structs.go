package api

type providerInfoResponseStruct struct {
	Results map[string]providerInfoItemStruct `json:"results"`
}

type providerInfoItemStruct struct {
	Available   bool    `json:"available"`
	Weight      float64 `json:"weight"`
	LastUpdated int64   `json:"last_updated"`
}
