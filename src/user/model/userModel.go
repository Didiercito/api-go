package usuarios

type Usuario struct {
	CLVE_CLIENTE    int    `json:"CLVE_CLIENTE"`
	NOMBRE_COMPLETO string `json:"NOMBRE_COMPLETO"`
	CELULAR         string `json:"CELULAR"`
	EMAIL           string `json:"EMAIL"`
}
