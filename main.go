package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

var (
	router *mux.Router
)

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
	router = mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("Hello World!"))
	}).Methods("GET")
	log.Println("Router Loaded")
}

func startServer() {
	addr := viper.GetString("SERVER_PORT")

	srv := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: viper.GetDuration("SERVER_WRITE_TIMEOUT"),
		ReadTimeout:  viper.GetDuration("SERVER_READ_TIMEOUT"),
	}

	log.Printf("Server running on: %s", addr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
