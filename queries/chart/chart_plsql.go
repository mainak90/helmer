package chartQueries

import (
	"database/sql"
	"github.com/mainak90/helmer/models"
	"log"
	"strings"
)

type ChartQueries struct {
}

func logFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Getting chart list for postgresql database
func (b ChartQueries) GetCharts(db *sql.DB, chart models.Chart, charts []models.Chart) []models.Chart {
	rows, err := db.Query("select * from charts")
	logFatal(err)

	for rows.Next() {
		err := rows.Scan(&chart.ID, &chart.Name, &chart.Version, &chart.Path)

		if err == sql.ErrNoRows {
			log.Printf("No rows found!!")
			return nil
		}

		if err != nil && err != sql.ErrNoRows {
			logFatal(err)
		}

		charts = append(charts, chart)
	}

	defer rows.Close()

	return charts
}

// Getting a specific chart from postgresql database, not needed for now, can be used later.
func (b ChartQueries) GetChart(db *sql.DB, chart models.Chart, id int) models.Chart {
	rows := db.QueryRow("select * from charts where id=$1", id)

	err := rows.Scan(&chart.ID, &chart.Name, &chart.Version, &chart.Path)
	logFatal(err)

	return chart
}

// Outlays the database action after a new chart is added
func (b ChartQueries) AddChart(db *sql.DB, chart models.Chart) int {
	err := db.QueryRow("insert into charts (name, version, path) values($1, $2, $3) RETURNING id;",
		chart.Name, chart.Version, chart.Path).Scan(&chart.ID)

	logFatal(err)

	return chart.ID
}

// Incase chart is updated, implementation is this one, but yet not executed upon.
func (b ChartQueries) UpdateChart(db *sql.DB, chart models.Chart) int64 {
	result, err := db.Exec("update charts set Name=$1, Version=$2, Path=$3 where id=$4 RETURNING id",
		&chart.Name, &chart.Version, &chart.Path, &chart.ID)

	logFatal(err)

	rowsUpdated, err := result.RowsAffected()
	logFatal(err)

	return rowsUpdated
}

// Remove the chart from the database, used by the chart watcher helper method to act upon if chart is deleted manually from filesystem.
func (b ChartQueries) RemoveChart(db *sql.DB, name string, version string) int64 {
	result, err := db.Exec("DELETE FROM charts WHERE name=$1 AND version=$2;", name, version)

	if err == sql.ErrNoRows {
		log.Printf("No rows found!!")
		return 0
	}

	if err != nil && err != sql.ErrNoRows {
		logFatal(err)
	}

	rowsDeleted, err := result.RowsAffected()

	logFatal(err)

	return rowsDeleted
}

// To supply the chart path is FS to the addChart method to pick charts from using chart.Load
func (b ChartQueries) GetChartPath(db *sql.DB, chart models.Chart, name string, version string) string {
	rows := db.QueryRow("select path from charts where name=$1 and version=$2", name, version)
	err := rows.Scan(&chart.Path)

	if err == sql.ErrNoRows {
		log.Printf("No rows found for chart %+v with version %+s sending NotFound response.", name, version)
		notfound := "Notfound"
		return notfound
	}

	if err != nil && err != sql.ErrNoRows {
		logFatal(err)
	}
	return chart.Path
}

// Add entry to add the chart release record into the database table.
func (b ChartQueries) AddDeploy(db *sql.DB, deploy models.Deploy) (int, error) {
	vars := strings.Join(deploy.Vars, ",")
	err := db.QueryRow("insert into deploys (deploymentName, deploymentDate, chartName, chartVersion, namespace, valuesOverrided, state) values($1, $2, $3, $4, $5, $6, $7) RETURNING id;",
		deploy.Name, deploy.Time, deploy.Chart, deploy.Version, deploy.Namespace, vars, deploy.Status).Scan(&deploy.ID)

	if err != nil {
		log.Printf("Error encountered: %s", err)
		return 0, err
	}

	return deploy.ID, nil
}

// Fetch deployment list from the table
func (b ChartQueries) GetDeploys(db *sql.DB, deploy models.Deploy, deploys []models.Deploy) []models.Deploy {
	rows, err := db.Query("select id, deploymentName, deploymentDate, chartName, chartVersion, namespace, state from deploys")
	logFatal(err)

	for rows.Next() {
		err := rows.Scan(&deploy.ID, &deploy.Name, &deploy.Time, &deploy.Chart, &deploy.Version, &deploy.Namespace, &deploy.Status)

		if err == sql.ErrNoRows {
			log.Printf("No rows found!!")
			return nil
		}

		if err != nil && err != sql.ErrNoRows {
			logFatal(err)
		}

		deploys = append(deploys, deploy)
	}

	defer rows.Close()

	return deploys
}

// Incase a deploy does not exist or manually deleted directly from helm cli or from kubernetes, delete the release ref from the database.
func (b ChartQueries) RemoveDeployment(db *sql.DB, name string) (int64, error) {
	result, err := db.Exec("delete from deploys where name = $1;", name)
	logFatal(err)

	rowsDeleted, err := result.RowsAffected()

	if err == sql.ErrNoRows {
		log.Printf("No rows found!!")
		return 0, err
	}

	if err != nil && err != sql.ErrNoRows {
		logFatal(err)
	}

	return rowsDeleted, nil
}
