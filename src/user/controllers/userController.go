package controllers
import (
	"encoding/json"
	"go-api/src/db"
	"go-api/src/rabbitmq"
	usuarios "go-api/src/user/model"
	"go-api/src/services"

	"log"
	"net/http"
	"regexp"
	"time"
	"strconv"
	"github.com/gorilla/mux"
)

var (
	validNombre  = regexp.MustCompile(`^[a-zA-Z\s]+$`).MatchString
	validCelular = regexp.MustCompile(`^\d+$`).MatchString
	validEmail   = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString
)

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func GetUsuarios(w http.ResponseWriter, r *http.Request) {
	conn := db.GetDB()
	rows, err := conn.Query("SELECT CLVE_CLIENTE, NOMBRE_COMPLETO, CELULAR, EMAIL FROM usuarios")
	if err != nil {
		http.Error(w, "Error al obtener usuarios", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []usuarios.Usuario
	for rows.Next() {
		var u usuarios.Usuario
		if err := rows.Scan(&u.CLVE_CLIENTE, &u.NOMBRE_COMPLETO, &u.CELULAR, &u.EMAIL); err != nil {
			http.Error(w, "Error al escanear usuarios", http.StatusInternalServerError)
			return
		}
		lista = append(lista, u)
	}

	w.Header().Set("Content-Type", "application/json")

	response := Response{
		Status:  "success",
		Message: "Usuarios obtenidos",
		Data:    lista,
	}
	json.NewEncoder(w).Encode(response)
}

func GetUsuario(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	idStr := params["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	conn := db.GetDB()
	var u usuarios.Usuario
	err = conn.QueryRow("SELECT CLVE_CLIENTE, NOMBRE_COMPLETO, CELULAR, EMAIL FROM usuarios WHERE CLVE_CLIENTE = $1", id).
		Scan(&u.CLVE_CLIENTE, &u.NOMBRE_COMPLETO, &u.CELULAR, &u.EMAIL)
	if err != nil {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func SearchUsuarios(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    campo := query.Get("campo")
    valor := query.Get("valor")

    var req usuarios.SearchRequest
    req.SearchID = services.GenerateUUID()

    switch campo {
    case "CLVE_CLIENTE":
        clave, err := strconv.Atoi(valor)
        if err != nil {
            respondWithError(w, http.StatusBadRequest, "ID inválido")
            return
        }
        req.CLVE_CLIENTE = clave
    case "NOMBRE_COMPLETO":
        if !validNombre(valor) {
            respondWithError(w, http.StatusBadRequest, "Nombre con caracteres inválidos")
            return
        }
        req.NOMBRE_COMPLETO = valor
    case "EMAIL":
        if !validEmail(valor) {
            respondWithError(w, http.StatusBadRequest, "Email con formato inválido")
            return
        }
        req.EMAIL = valor
    case "CELULAR":
        if !validCelular(valor) {
            respondWithError(w, http.StatusBadRequest, "Celular con formato inválido")
            return
        }
        req.CELULAR = valor
    default:
        respondWithError(w, http.StatusBadRequest, "Campo de búsqueda inválido")
        return
    }

    err := rabbitmq.PublishSearchRequest(req)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Error enviando búsqueda a cola")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status":   "success",
        "message":  "Búsqueda enviada a procesamiento",
        "searchID": req.SearchID,
    })
}

func GetSearchResults(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	searchID := query.Get("search_id")
	if searchID == "" {
		respondWithError(w, http.StatusBadRequest, "search_id es requerido")
		return
	}

	resultsChan := make(chan []usuarios.Usuario)
	err := rabbitmq.ConsumeSearchResults(searchID, resultsChan)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error al iniciar consumo de resultados")
		return
	}

	select {
	case resultados := <-resultsChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "Resultados obtenidos",
			Data:    resultados,
		})
	case <-time.After(10 * time.Second):
		respondWithError(w, http.StatusRequestTimeout, "Tiempo de espera agotado para resultados")
	}
}



func CreateUsuario(w http.ResponseWriter, r *http.Request) {
	var u usuarios.Usuario
	log.Printf("📝 Request recibido en /api/v1/crear/usuarios")

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		log.Printf("❌ Error parsing JSON: %v", err)
		http.Error(w, "JSON inválido o CLVE_CLIENTE no es numérico", http.StatusBadRequest)
		return
	}

	log.Printf("✅ JSON parseado: CLVE_CLIENTE=%d, NOMBRE=%s", u.CLVE_CLIENTE, u.NOMBRE_COMPLETO)

	if u.CLVE_CLIENTE <= 0 {
		log.Printf("❌ CLVE_CLIENTE inválido: %d", u.CLVE_CLIENTE)
		http.Error(w, "CLVE_CLIENTE debe ser positivo", http.StatusBadRequest)
		return
	}

	u.ERRORES = make(map[string]string)

	if !validNombre(u.NOMBRE_COMPLETO) {
		u.ERRORES["NOMBRE_COMPLETO"] = "nombre_con_caracteres_invalidos"
	}
	if !validCelular(u.CELULAR) {
		u.ERRORES["CELULAR"] = "formato_invalido"
	}
	if !validEmail(u.EMAIL) {
		u.ERRORES["EMAIL"] = "formato_invalido"
	}

	// Aquí corregimos: enviamos un slice con un solo usuario
	if err := rabbitmq.PublishUsuariosChunk([]usuarios.Usuario{u}); err != nil {
		log.Printf("❌ Error enviando a RabbitMQ: %v", err)
		http.Error(w, "Error al procesar usuario", http.StatusInternalServerError)
		return
	}

	log.Printf("🚀 Usuario %d enviado a RabbitMQ con posibles errores", u.CLVE_CLIENTE)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	response := Response{
		Status:  "accepted_with_errors",
		Message: "Usuario enviado para procesamiento, pero con errores que deben corregirse",
		Data:    u,
	}
	json.NewEncoder(w).Encode(response)
}

func UpdateUsuario(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	idStr := params["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var u usuarios.Usuario
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	u.CLVE_CLIENTE = int(id)

	u.ERRORES = make(map[string]string)

	if !validNombre(u.NOMBRE_COMPLETO) {
		u.ERRORES["NOMBRE_COMPLETO"] = "nombre_con_caracteres_invalidos"
	}
	if !validCelular(u.CELULAR) {
		u.ERRORES["CELULAR"] = "formato_invalido"
	}
	if !validEmail(u.EMAIL) {
		u.ERRORES["EMAIL"] = "formato_invalido"
	}

	conn := db.GetDB()
	res, err := conn.Exec("UPDATE usuarios SET NOMBRE_COMPLETO=$1, CELULAR=$2, EMAIL=$3 WHERE CLVE_CLIENTE=$4",
		u.NOMBRE_COMPLETO, u.CELULAR, u.EMAIL, id)
	if err != nil {
		http.Error(w, "Error al actualizar usuario", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	// Enviar el usuario como slice con un solo elemento
	if err := rabbitmq.PublishUsuariosChunk([]usuarios.Usuario{u}); err != nil {
		log.Printf("❌ Error enviando a RabbitMQ: %v", err)
		http.Error(w, "Error al notificar actualización", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := Response{
		Status:  "success",
		Message: "Usuario actualizado y enviado para procesamiento",
		Data:    u,
	}
	json.NewEncoder(w).Encode(response)
}

func DeleteUsuario(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	idStr := params["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	conn := db.GetDB()
	res, err := conn.Exec("DELETE FROM usuarios WHERE CLVE_CLIENTE = $1", id)
	if err != nil {
		http.Error(w, "Error al eliminar usuario", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Status:  "success",
		Message: "Usuario eliminado",
	}
	json.NewEncoder(w).Encode(response)
}

func BulkUploadUsuarios(w http.ResponseWriter, r *http.Request) {
    file, _, err := r.FormFile("archivo")
    if err != nil {
        http.Error(w, "Archivo no enviado o inválido", http.StatusBadRequest)
        return
    }
    defer file.Close()

    usuariosList, err := services.ParseExcel(file)
    if err != nil {
        http.Error(w, "Error al procesar archivo: "+err.Error(), http.StatusBadRequest)
        return
    }

    const batchSize = 5000
    for i := 0; i < len(usuariosList); i += batchSize {
        end := i + batchSize
        if end > len(usuariosList) {
            end = len(usuariosList)
        }
        lote := usuariosList[i:end]

        if err := rabbitmq.PublishUsuariosChunk(lote); err != nil {
            http.Error(w, "Error enviando lote a RabbitMQ: "+err.Error(), http.StatusInternalServerError)
            return
        }
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(Response{
        Status:  "accepted",
        Message: "Archivo recibido y usuarios enviados en lotes para procesamiento",
    })
}