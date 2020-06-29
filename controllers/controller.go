package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mainak90/helmer/models"
	chartQueries "github.com/mainak90/helmer/queries/chart"
	"github.com/mainak90/helmer/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Self explanatory, does multi-part upload of helm archives with the associated index.html
func UploadHelmChart(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("File Upload Endpoint Hit")

		if r.Method != "POST" {
			log.Println("This endpoint only supports POST request!")
			return
		}

		r.ParseMultipartForm(50 << 20)

		file, handler, err := r.FormFile("myFile")

		if filepath.Ext(handler.Filename) != ".tgz" {
			log.Println("Error encountered! The provided file extension is not .tgz")
			return
		}

		if err != nil {
			log.Println("Error Retrieving the File")
			log.Fatal(err)
			return
		}

		defer file.Close()

		log.Printf("Uploaded File: %+v\n", handler.Filename)

		log.Printf("File Size: %+v\n", handler.Size)

		log.Printf("MIME Header: %+v\n", handler.Header)

		// The assumption is that the filename will always follow the format <app>-<version>.tgz
		// the app seperators should be '_' and not '-'
		withoutExt := strings.Split(handler.Filename, ".tgz")

		split := strings.Split(withoutExt[0], "-")

		chartName := split[0]

		version := split[1]

		// Creates the directories if it doesn't exist
		os.MkdirAll("/tmp/charts/"+chartName+"/"+version, os.ModePerm)

		// Creates the file descriptor
		f, err := os.OpenFile("/tmp/charts/"+chartName+"/"+version+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)

		if err != nil {
			log.Fatal(err)
			return
		}

		defer f.Close()

		io.Copy(f, file)

		log.Printf("File uploaded successfully: %+v\n", chartName)

		log.Printf("Chart Version: %+v\n", version)

		charted := models.Chart{Name: chartName, Version: version, Path: "/tmp/charts/" + chartName + "/" + version + "/" + handler.Filename}

		// Call the sql query struct
		chartQuery := chartQueries.ChartQueries{}

		// Add the chart into the sql table
		chartQuery.AddChart(db, charted)
	}
}

// List helm charts from postgresql
func ListHelmCharts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Charts Listing Endpoint Hit")

		var chart models.Chart

		charts := []models.Chart{}

		if r.Method != "GET" {
			log.Println("This endpoint only supports GET request!")
			return
		}

		chartQuery := chartQueries.ChartQueries{}

		charts = chartQuery.GetCharts(db, chart, charts)

		json.NewEncoder(w).Encode(charts)
	}
}

// Deploy the release/chart into the associated kubernetes cluster
//{"name":"redis","chart":"redis-","version":"0.5.7","namespace": "default","vars": ["mysqlRootPassword=admin@123,imagePullPolicy=IfNotPresent"]}
func DeployApp(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Endpoint hit!")
		var deploys int
		var deploy models.Deploy
		var chart models.Chart

		err := json.NewDecoder(r.Body).Decode(&deploy)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse the vars from loaded struct
		name := deploy.Chart

		version := deploy.Version

		namespace := deploy.Namespace

		log.Printf("Namespace is %+v\n", namespace)
		log.Printf("Name is %+v\n", name)
		log.Printf("Version is %+v\n", version)

		chartQuery := chartQueries.ChartQueries{}

		chartPath := chartQuery.GetChartPath(db, chart, name, version)

		log.Printf("Chart path %+v\n", chartPath)
		// Chart path sends NotFound incase chart is not found, loop exexution to stop in such case.
		if chartPath == "Notfound" {
			log.Printf("No chart path is retrieved from the provided data, cannot proceed...")
			return
		}

		log.Printf("Deploying chart %-s version %-v into namespace %-v\n", name, version, namespace)

		// Rertieve client config
		actionConfig, err := utils.GetActionConfig(namespace)

		if err != nil {
			panic(err)
		}


		iCli := action.NewInstall(actionConfig)

		iCli.ReleaseName = deploy.Name

		// Load chart..
		charted, err := loader.Load(chartPath)

		if err != nil {
			log.Printf("Error encountered: %-v\n", err)
			return
		}

		// Check if chart is valid before installing..
		validInstallableChart, err := utils.IsChartInstallable(charted)

		// Stop here if chart is not valid..
		if !validInstallableChart {
			log.Printf("Error encountered: %-v\n", err)
			return
		}

		// The way i see online(due to lack of proper documentation) ,
		vals := make(map[string]interface{})
		for _, value := range deploy.Vars {
			if err := strvals.ParseInto(value, vals); err != nil {
				fmt.Errorf("failed parsing set data")
				return
			}
		}
		log.Printf("Loaded chart %-s with version %-v\n", name, version,)
		log.Printf("Loaded chart values as --set : %s\n", vals)
		log.Printf("Trying to deploy chart %-s version %-v into namespace %-v\n", name, version, namespace)

		rel, err := iCli.Run(charted, vals)
		if err != nil {
			log.Printf("Error encountered : %-v\n",err)
			deploy.Status = "Failed"
			deploy.Time = time.Now().Unix()
			deploys, err = chartQuery.AddDeploy(db, deploy)

			if err != nil {
				log.Printf("Failed to add row into table deployment for chart %-s version %-v for namespace %s\n", name, version, namespace)
			}

			json.NewEncoder(w).Encode(deploys)
			return
		}

		log.Printf("Successfully installed chart-release : %-v\n", rel.Name)
		deploy.Status = "Success"
		deploy.Time = time.Now().Unix()
		// Add record as deployed chart into the database.
		log.Printf("Trying to add entry into database")

		deploys, err = chartQuery.AddDeploy(db, deploy)

		if err != nil {
			log.Printf("Failed to add row into table deployment for chart %s version %s for namespace %s\n", name, version, namespace)
			return
		}

		log.Printf("Added row into table deployment for chart %s version %s for namespace %s\n", name, version, namespace)

		json.NewEncoder(w).Encode(deploys)
	}
}

func ListDeployments(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Helm Deployment Listing Endpoint Hit")

		var deploy models.Deploy

		deploys := []models.Deploy{}

		if r.Method != "GET" {
			log.Println("This endpoint only supports GET request!")
			return
		}

		chartQuery := chartQueries.ChartQueries{}

		deploys = chartQuery.GetDeploys(db, deploy, deploys)

		json.NewEncoder(w).Encode(deploys)
	}
}

func DeleteDeployment(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Println("Helm Deployment Deletion Endpoint Hit")

		if r.Method != "POST" {
			log.Println("This endpoint only supports POST request!")
			return
		}

		params := mux.Vars(r)

		releaseName := params["name"]

		namespace := params["namespace"]

		if utils.IsInstalled(namespace, releaseName) {
			log.Printf("Release %s is  installed in namespace %s proceed to undeploy \n", releaseName, namespace)
		} else {
			log.Printf("Release %s is not installed in namespace %s \n", releaseName, namespace)
			return
		}


		actionConfig, err := utils.GetActionConfig(namespace)

		if err != nil {
			panic(err)
		}

		iCli := action.NewUninstall(actionConfig)

		rel, err := iCli.Run(releaseName)

		if err != nil {
			log.Printf("Error encountered : %-v\n",err)
		}

		log.Printf("Successfully uninstalled chart-release : %-v\n", rel.Release.Name)

		chartQuery := chartQueries.ChartQueries{}

		log.Printf("Removing record for chart release: %-v from database table \n", rel.Release.Name)

		rowdel, err := chartQuery.RemoveDeployment(db, releaseName)

		if err != nil {
			log.Printf("Error encountered %-v\n", err)
			return
		}

		log.Printf("Removed record for release %-v in namespace %-s from database .. \n", rel.Release.Name, rel.Release.Namespace)

		json.NewEncoder(w).Encode(rowdel)
	}
}