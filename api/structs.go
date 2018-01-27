package api

type providerInfoResponseStruct struct {
	Results []providerInfoItemStruct `json:"results"`
}

func (pi providerInfoResponseStruct) Len() int {
	return len(pi.Results)
}

func (pi providerInfoResponseStruct) Swap(i, j int) {
	arr := pi.Results
	arr[i], arr[j] = arr[j], arr[i]
}

func (pi providerInfoResponseStruct) Less(i, j int) bool {
	return pi.Results[i].Name < pi.Results[j].Name
}

type providerInfoItemStruct struct {
	Name        string  `json:"name"`
	Available   bool    `json:"available"`
	Weight      float64 `json:"weight"`
	LastUpdated int64   `json:"last_updated"`
}
