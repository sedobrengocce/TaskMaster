package main

import (
	"log"
	"github.com/sedobrengocce/TaskMaster/internal/server" 
)

func main() {
	// Crea e avvia il server
	err := server.Run()
	if err != nil {
		log.Fatalf("Errore durante l'avvio del server: %v", err)
	}
}
