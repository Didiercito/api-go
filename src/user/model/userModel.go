package usuarios

type Usuario struct {
	CLVE_CLIENTE    int    `json:"CLVE_CLIENTE"`
	NOMBRE_COMPLETO string `json:"NOMBRE_COMPLETO"`
	CELULAR         string `json:"CELULAR"`
	EMAIL           string `json:"EMAIL"`
	ERRORES         map[string]string `json:"errores,omitempty"`
	SearchID        string `json:"search_id,omitempty"`
}


type SearchRequest struct {
	SearchID        string `json:"search_id"`
	CLVE_CLIENTE    int    `json:"CLVE_CLIENTE"`
	NOMBRE_COMPLETO string `json:"NOMBRE_COMPLETO"`
	CELULAR         string `json:"CELULAR"`
	EMAIL           string `json:"EMAIL"`
}
