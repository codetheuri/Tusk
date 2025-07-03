package main

import (
	"log"
	"net/http"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/router"
)

func main() {
	config.InitDb()
	router.SetupRouter()
	log.Println("Server running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
