// Utilidad de desarrollo: genera el hash bcrypt de una contraseña para poder
// sembrar usuarios a mano mientras no exista el endpoint de registro.
//
//	go run ./cmd/hashpw "mi-contraseña"
package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("uso: go run ./cmd/hashpw <contraseña>")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("no se pudo generar el hash: %v", err)
	}

	fmt.Println(string(hash))
}
