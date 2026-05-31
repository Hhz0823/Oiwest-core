package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Hhz0823/oiwest-core/O-ui/database"
	"github.com/Hhz0823/oiwest-core/O-ui/web"
)

var (
	port   = flag.Int("port", 54321, "panel listen port")
	dbPath = flag.String("db", "o-ui.db", "database path")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fmt.Println("============================================")
	fmt.Println("  O-ui v1.0.0 - Oiwest Core Management Panel")
	fmt.Println("============================================")

	if err := database.Init(*dbPath); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer database.Close()
	log.Printf("[O-ui] Database initialized: %s", *dbPath)

	if err := web.StartServer(*port); err != nil {
		log.Fatalf("Server failed: %v", err)
		os.Exit(1)
	}
}
