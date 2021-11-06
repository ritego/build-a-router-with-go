package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ritego/build-a-router-with-go/router"
	"github.com/spf13/viper"
)

var rr = router.New()

func main() {
	initConfig()
	setupRouter()
	startServer()
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error reading env file: %w", err))
	}
	viper.AutomaticEnv()
	viper.WatchConfig()
	log.Println("Config Loaded")
}

func setupRouter() {

	rr.HandleFunc("GET:/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("Root - Hello World!"))
	})

	rr.HandleFunc("GET:/path-one", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("Path One - Hello World!"))
	})

	rr.HandleFunc("GET:/path-one/path-two", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("Path Two - Hello World!"))
	})

	log.Println("Router Loaded")
}

func startServer() {
	addr := viper.GetString("SERVER_PORT")

	srv := &http.Server{
		Handler:      rr,
		Addr:         addr,
		WriteTimeout: viper.GetDuration("SERVER_WRITE_TIMEOUT"),
		ReadTimeout:  viper.GetDuration("SERVER_READ_TIMEOUT"),
	}

	log.Printf("Server running on: %s", addr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
