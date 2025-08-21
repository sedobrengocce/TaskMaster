package main

import (
	"database/sql"
	"log"

	"github.com/sedobrengocce/TaskMaster/internal/env"
	"github.com/sedobrengocce/TaskMaster/internal/server"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
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

	srv := server.NewServer(conn)
	if err := srv.Run(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	log.Println("Server started successfully on port 3000")
}
