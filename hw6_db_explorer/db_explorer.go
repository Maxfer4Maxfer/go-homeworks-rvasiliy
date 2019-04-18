package main

import (
	"database/sql"
	"net/http"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

func NewDbExplorer(dbConn *sql.DB) (http.Handler, error) {

	db := &database{
		conn: dbConn,
	}

	err := db.refreshDatabaseStructure()
	if err != nil {
		panic(err)
	}

	api := &databaseAPI{
		db: db,
	}

	router := newRouter()
	router.setHandler(api.getAllTables, "/", "GET")
	router.setHandler(api.getRows, "/table", "GET")
	router.setHandler(api.getRowByID, "/table/id", "GET")
	router.setHandler(api.addRow, "/table", "PUT")
	router.setHandler(api.updateRow, "/table/id", "POST")
	router.setHandler(api.deleteRow, "/table/id", "DELETE")

	return router, nil
}
