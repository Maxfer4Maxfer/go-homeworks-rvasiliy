package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/maxfer4maxfer/goDebuger"
)

func getTableFromDatabase(w http.ResponseWriter, r *http.Request, db *database, tName string) *table {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	table, err := db.getTable(tName)

	if err != nil && strings.Contains(err.Error(), "There is no table with the name") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		data := make(map[string]string)
		data["error"] = "unknown table"
		errJSON, _ := json.Marshal(data)
		w.Write(errJSON)
		return nil
	}

	if err != nil {
		fmt.Fprintln(w, err)
		return nil
	}

	return table
}

type databaseAPI struct {
	db *database
}

func (api *databaseAPI) getAllTables(w http.ResponseWriter, r *http.Request) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	tables, err := api.db.getAllTables()
	if err != nil {
		fmt.Fprintln(w, err)
	}

	data := make(map[string]map[string][]string)
	data["response"] = make(map[string][]string)
	data["response"]["tables"] = make([]string, 0)
	for _, t := range tables {
		data["response"]["tables"] = append(data["response"]["tables"], t)
	}

	// data -> json
	resultJSON, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resultJSON)

}

func (api *databaseAPI) getRows(w http.ResponseWriter, r *http.Request) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// get input parameters
	tName := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 5
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		offset = 0
	}

	// find a table
	table := getTableFromDatabase(w, r, api.db, tName)
	if table == nil {
		return
	}

	// get all rows
	rows, err := table.getRows(limit, offset)
	if err != nil {
		fmt.Fprintln(w, err)
	}

	// convert rows to json applicabale data structure
	data := make(map[string]map[string][]map[string]interface{})
	data["response"] = make(map[string][]map[string]interface{})
	data["response"]["records"] = make([]map[string]interface{}, 0)
	for _, row := range rows {
		dataRow := make(map[string]interface{})
		for _, c := range row.cells {
			switch c.cellType {
			case "int":
				dataRow[c.colName] = c.value.(int)
			case "string":
				switch c.value {
				case nil:
					dataRow[c.colName] = c.value
				default:
					dataRow[c.colName] = c.value.(string)
				}
			}
		}
		data["response"]["records"] = append(data["response"]["records"], dataRow)
	}

	// convert to JSON
	resultJSON, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// write output
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resultJSON)
}

func (api *databaseAPI) getRowByID(w http.ResponseWriter, r *http.Request) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// get input parameters
	tName := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
	tID, err := strconv.Atoi(strings.Split(strings.Trim(r.URL.Path, "/"), "/")[1])
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// find a table
	table := getTableFromDatabase(w, r, api.db, tName)
	if table == nil {
		return
	}

	row, err := table.getRowByID(tID)

	if err != nil && strings.Contains(err.Error(), "Can't get a row with id") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		data := make(map[string]string)
		data["error"] = "record not found"
		errJSON, _ := json.Marshal(data)
		w.Write(errJSON)
		return
	}

	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// convert a row to json applicabale data structure
	data := make(map[string]map[string]map[string]interface{})
	data["response"] = make(map[string]map[string]interface{})
	data["response"]["record"] = make(map[string]interface{})
	for _, c := range row.cells {
		switch c.cellType {
		case "int":
			data["response"]["record"][c.colName] = c.value.(int)
		case "string":
			switch c.value {
			case nil:
				data["response"]["record"][c.colName] = c.value
			default:
				data["response"]["record"][c.colName] = c.value.(string)
			}
		}
	}

	// convert to JSON
	resultJSON, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	// write output
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resultJSON)
}

func (api *databaseAPI) addRow(w http.ResponseWriter, r *http.Request) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	tName := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]

	// find a table
	table := getTableFromDatabase(w, r, api.db, tName)
	if table == nil {
		return
	}

	defer r.Body.Close()

	var input interface{}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &input)

	in, _ := input.(map[string]interface{})

	putValues := make(map[string]interface{}, 0)
	for _, col := range table.columns {
		if !col.pk {
			v, ok := in[col.name]
			if ok {
				putValues[col.name] = v
			}
			if !ok && col.Null {
				putValues[col.name] = nil
			}
			// default values
			if !ok && !col.Null {
				switch col.colType {
				case "text":
					fallthrough
				case "varchar(255)":
					putValues[col.name] = ""
				case "int":
					putValues[col.name] = 0
				}

			}
		}
	}

	id, err := table.addRow(putValues)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data := make(map[string]map[string]int64)
	data["response"] = make(map[string]int64)
	data["response"][table.getPrimaryKey()] = id
	respJSON, _ := json.Marshal(data)
	w.Write(respJSON)

}

func (api *databaseAPI) updateRow(w http.ResponseWriter, r *http.Request) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// get input parameters
	tName := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
	tID, err := strconv.Atoi(strings.Split(strings.Trim(r.URL.Path, "/"), "/")[1])
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// find a table
	table := getTableFromDatabase(w, r, api.db, tName)
	if table == nil {
		return
	}

	defer r.Body.Close()

	var input interface{}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &input)

	in, _ := input.(map[string]interface{})

	putValues := make(map[string]interface{}, 0)
	for _, col := range table.columns {
		v, ok := in[col.name]
		if !col.pk && ok && v != nil {
			// validate type of v and col.Type
			matchType := false
			if col.colType == "int" && reflect.TypeOf(v).Name() == "float64" {
				matchType = true
			}
			if (col.colType == "text" || col.colType == "varchar(255)") && reflect.TypeOf(v).Name() == "string" {
				matchType = true
			}

			if matchType {
				putValues[col.name] = v
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				data := make(map[string]string)
				data["error"] = fmt.Sprintf("field %v have invalid type", col.name)
				errJSON, _ := json.Marshal(data)
				w.Write(errJSON)
				return
			}
		}
		if !col.pk && ok && v == nil && col.Null {
			putValues[col.name] = v
		}

		if !col.pk && ok && v == nil && !col.Null {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			data := make(map[string]string)
			data["error"] = fmt.Sprintf("field %v have invalid type", col.name)
			errJSON, _ := json.Marshal(data)
			w.Write(errJSON)
			return
		}

		if col.pk && ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			data := make(map[string]string)
			data["error"] = fmt.Sprintf("field %v have invalid type", col.name)
			errJSON, _ := json.Marshal(data)
			w.Write(errJSON)
			return
		}
	}

	id, err := table.updateRow(tID, putValues)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data := make(map[string]map[string]int64)
	data["response"] = make(map[string]int64)
	data["response"]["updated"] = id
	respJSON, _ := json.Marshal(data)
	w.Write(respJSON)

}

func (api *databaseAPI) deleteRow(w http.ResponseWriter, r *http.Request) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// get input parameters
	tName := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
	tID, err := strconv.Atoi(strings.Split(strings.Trim(r.URL.Path, "/"), "/")[1])
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// find a table
	table := getTableFromDatabase(w, r, api.db, tName)
	if table == nil {
		return
	}

	resp, err := table.deleteRowByID(tID)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data := make(map[string]map[string]int64)
	data["response"] = make(map[string]int64)
	data["response"]["deleted"] = resp
	respJSON, _ := json.Marshal(data)
	w.Write(respJSON)
}
