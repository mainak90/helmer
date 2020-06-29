package main

import (
	"database/sql"
	_ "encoding/json"
	"github.com/mainak90/helmer/controllers"
	"github.com/mainak90/helmer/driver"
	"github.com/mainak90/helmer/models"
	"github.com/mainak90/helmer/utils"
	"log"
	"net/http"
	_ "strconv"

	"github.com/gorilla/mux"
	"github.com/subosito/gotenv"
)

var db *sql.DB

var charts []models.Chart

func init() {
	gotenv.Load(".env")
}

func logFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type WithCORS struct {
	r *mux.Router
}

func (s *WithCORS) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		res.Header().Set("Access-Control-Allow-Origin", origin)
		res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		res.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}

	// Stop here for a Preflighted OPTIONS request.
	if req.Method == "OPTIONS" {
		return
	}
	// Lets Gorilla work
	s.r.ServeHTTP(res, req)
}

func main() {
	db = driver.ConnectDB()
	router := mux.NewRouter()
	log.Println("Adding chartUpload endpoint...")
	router.HandleFunc("/uploadChart", controllers.UploadHelmChart(db)).Methods("POST")
	log.Println("Adding listChart endpoint...")
	router.HandleFunc("/getChartList", controllers.ListHelmCharts(db)).Methods("GET")
	log.Println("Adding deployChart endpoint...")
	router.HandleFunc("/deployChart", controllers.DeployApp(db)).Methods("POST")
	log.Println("Adding listHelmDeployments endpoint...")
	router.HandleFunc("/getDeploymentList", controllers.ListDeployments(db)).Methods("GET")
	log.Println("Adding deleteHelmDeployments endpoint...")
	router.HandleFunc("/deleteDeployment/namespace/{namespace}/name/{name}", controllers.DeleteDeployment(db)).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	go func() {
		utils.WatchFile(db)
	}()
	go func() {
		log.Println("Starting server...")
		log.Fatal(http.ListenAndServe(":8900", &WithCORS{router}))
	}()
	select {}
}
