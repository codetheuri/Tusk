package main

import (
	"log"
	"net/http"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/app/routers"
)

func main() {
	// config.InitDb()
	config.LoadConfig()
	router.SetupRouter()
	log.Println("Server running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
