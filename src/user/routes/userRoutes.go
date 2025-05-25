package routes

import (
	"go-api/src/user/controllers"
	"github.com/gorilla/mux"
)

func RegistrarUsuarioRoutes(r *mux.Router) {
    r.HandleFunc("/usuarios", controllers.GetUsuarios).Methods("GET")
    r.HandleFunc("/usuarios/{id}", controllers.GetUsuario).Methods("GET")
    r.HandleFunc("/crear/usuarios", controllers.CreateUsuario).Methods("POST", "OPTIONS")
    r.HandleFunc("/actualizar/usuarios/{id}", controllers.UpdateUsuario).Methods("PUT", "OPTIONS")
    r.HandleFunc("/eliminar/usuarios/{id}", controllers.DeleteUsuario).Methods("DELETE", "OPTIONS")
    r.HandleFunc("/usuarpos/upload", controllers.BulkUploadUsuarios).Methods("POST", "OPTIONS") 
    r.HandleFunc("/usuarpos/search", controllers.SearchUsuarios).Methods("POST", "OPTIONS")  
    r.HandleFunc("/usuarios/resultados", controllers.GetSearchResults).Methods("GET", "OPTIONS")
}

