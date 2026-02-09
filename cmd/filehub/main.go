package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kiry163/filehub/internal/api"
	"github.com/kiry163/filehub/internal/config"
	"github.com/kiry163/filehub/internal/db"
	"github.com/kiry163/filehub/internal/service"
	"github.com/kiry163/filehub/internal/storage"
	"github.com/kiry163/filehub/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version.String())
		return
	}

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o755); err != nil {
		log.Fatal(err)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatal(err)
	}

	minioStorage, err := storage.NewMinioStorage(context.Background(), cfg.Minio)
	if err != nil {
		log.Fatal(err)
	}

	svc := &service.Service{
		DB:      database,
		Storage: minioStorage,
		Config:  cfg,
	}

	router := api.NewRouter(svc)
	address := ":" + strconv.Itoa(cfg.Server.Port)
	log.Printf("filehub server listening on %s", address)
	if err := http.ListenAndServe(address, router); err != nil {
		log.Fatal(err)
	}
}
