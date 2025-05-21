package controllers

import (
	"encoding/json"
	"fmt"
	"go-api/src/db"
	usuarios "go-api/src/user/model"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
)

var (
	validNombre = regexp.MustCompile(`^[a-zA-Z\s]+$`).MatchString
	validCelular = regexp.MustCompile(`^\d+$`).MatchString
	validEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString
)

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
	json.NewEncoder(w).Encode(lista)
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

func CreateUsuario(w http.ResponseWriter, r *http.Request) {
    var u usuarios.Usuario
    if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
        http.Error(w, "JSON inválido o CLVE_CLIENTE no es numérico", http.StatusBadRequest)
        return
    }

    if !validNombre(u.NOMBRE_COMPLETO) {
        fmt.Println("⚠️ NOMBRE inválido:", u.NOMBRE_COMPLETO)
    }
    if !validCelular(u.CELULAR) {
        fmt.Println("⚠️ CELULAR inválido:", u.CELULAR)
    }
    if !validEmail(u.EMAIL) {
        fmt.Println("⚠️ EMAIL inválido:", u.EMAIL)
    }

    conn := db.GetDB()
    _, err := conn.Exec("INSERT INTO usuarios (CLVE_CLIENTE, NOMBRE_COMPLETO, CELULAR, EMAIL) VALUES ($1, $2, $3, $4)",
        u.CLVE_CLIENTE, u.NOMBRE_COMPLETO, u.CELULAR, u.EMAIL)
    if err != nil {
        http.Error(w, "Error al insertar usuario", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
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

    if !validNombre(u.NOMBRE_COMPLETO) {
        fmt.Println("⚠️ NOMBRE inválido:", u.NOMBRE_COMPLETO)
    }
    if !validCelular(u.CELULAR) {
        fmt.Println("⚠️ CELULAR inválido:", u.CELULAR)
    }
    if !validEmail(u.EMAIL) {
        fmt.Println("⚠️ EMAIL inválido:", u.EMAIL)
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

    w.WriteHeader(http.StatusOK)
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
	res, err := conn.Exec("DELETE FROM usuarios WHERE CLVE_CLIENTE=$1", id)
	if err != nil {
		http.Error(w, "Error al eliminar usuario", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}
