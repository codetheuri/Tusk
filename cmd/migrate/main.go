package migrate

import (
	"log"

	"github.com/codetheuri/todolist/config"
)


var databaseURl string

func init(){
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// databaseURl = cfg.DBUser + ":" + cfg.DBPass + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName
	databaseURl = cfg.DbURl
}