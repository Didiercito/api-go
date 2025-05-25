package services

import (
    "fmt"
    "mime/multipart"
    "strconv"

    "github.com/xuri/excelize/v2"
    usuarios "go-api/src/user/model"
)

func ParseExcel(file multipart.File) ([]usuarios.Usuario, error) {
    var usuariosSlice []usuarios.Usuario

    f, err := excelize.OpenReader(file)
    if err != nil {
        return nil, fmt.Errorf("error al abrir archivo Excel: %w", err)
    }

    sheetName := f.GetSheetName(0)
    if sheetName == "" {
        return nil, fmt.Errorf("no se encontró hoja en el archivo Excel")
    }

    rows, err := f.GetRows(sheetName)
    if err != nil {
        return nil, fmt.Errorf("error al obtener filas: %w", err)
    }

    for i, row := range rows {
        if i == 0 {
            continue
        }
        if len(row) < 4 {
            continue
        }

        clve, err := strconv.Atoi(row[0])
        if err != nil {
            continue 
        }

        usuario := usuarios.Usuario{
            CLVE_CLIENTE:    clve,
            NOMBRE_COMPLETO: row[1],
            CELULAR:        row[2],
            EMAIL:          row[3],
        }

        usuariosSlice = append(usuariosSlice, usuario)
    }

    return usuariosSlice, nil
}
