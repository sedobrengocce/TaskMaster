// in cmd/seeder/main.go
package main

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sedobrengocce/TaskMaster/internal/db" // Assicurati che il path sia corretto
	"github.com/sedobrengocce/TaskMaster/internal/env"
	"github.com/sedobrengocce/TaskMaster/internal/utils"
)

func main() {
	log.Println("Avvio dello script di seeding...")

	Env, err := env.ReadEnv(nil)
	if err != nil {
		log.Fatalf("Error reading environment variables: %v", err)
	}

	dbUrl := "mysql://" + Env.GetDBUser() + ":" + Env.GetDBPassword() + "@tcp(db:3306)/" + Env.GetDBName() + "?parseTime=true"
	conn, err := sql.Open("mysql", dbUrl)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer conn.Close()

	queries := db.New(conn)
	ctx := context.Background()

	mobileClientID := Env.GetMobileClientID()
	mobileSecret := Env.GetMobileSecret()

	hashedSecret, err := utils.HashString(mobileSecret)
	if err != nil {
		log.Fatalf("Errore nell'hashing del secret: %v", err)
	}

	err = queries.InsertClient(ctx, db.InsertClientParams{
		ClientID:         mobileClientID,
		ClientSecretHash: sql.NullString{String: hashedSecret, Valid: true},
		ClientType:       db.ClientsClientTypeConfidential,
		AppName:          sql.NullString{String: "Mobile App", Valid: true},
	})
	
	if err != nil {
		// Potrebbe fallire se l'utente esiste già (violazione UNIQUE), che è ok.
		log.Printf("Impossibile creare l'utente admin (potrebbe esistere già): %v\n", err)
	} else {
		log.Println("Utente admin creato/verificato con successo.")
	}
	
	log.Println("Seeding completato.")
}
