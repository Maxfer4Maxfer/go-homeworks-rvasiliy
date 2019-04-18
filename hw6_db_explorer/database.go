package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/maxfer4maxfer/goDebuger"
)

// --------------------- cell ---------------------
type cell struct {
	value    interface{}
	cellType string
	colName  string
}

func (c cell) String() string {
	return fmt.Sprintf("\t%v : %v\n", c.value, c.cellType)
}

func (c *cell) initiate(colName string, cellType string) error {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	switch cellType {
	case "INT":
		var varInt int
		c.value = &varInt
	case "VARCHAR", "TEXT":
		var varString sql.NullString
		c.value = &varString
	}
	c.colName = colName
	c.cellType = cellType
	return nil
}

func (c *cell) convertToGoType() error {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	switch c.cellType {
	case "INT":
		c.value = *(c.value.(*int))
		c.cellType = "int"
	case "VARCHAR", "TEXT":
		ns := *(c.value.(*sql.NullString))
		if ns.Valid {
			c.value = ns.String
		} else {
			c.value = nil
		}
		c.cellType = "string"
	}
	return nil
}

// --------------------- row ---------------------
type row struct {
	cells []*cell
}

func (r row) String() string {
	return fmt.Sprintf("%v", r.cells)
}

func (r *row) getCellValues() []interface{} {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	result := make([]interface{}, 0)
	for _, cell := range r.cells {
		result = append(result, cell.value)
	}
	return result
}

// --------------------- column ---------------------
type column struct {
	name    string
	colType string
	pk      bool
	Null    bool
}

func (c column) String() string {
	pk := ""
	if c.pk {
		pk = "*"
	}
	return fmt.Sprintf("\t%v%v[%v]\n", pk, c.name, c.colType)
}

// --------------------- table ---------------------
type table struct {
	db      *database
	name    string
	columns []*column
}

func (t table) String() string {
	return fmt.Sprintf("%v : \n\t%v\n ", t.name, t.columns)
}

func (t table) getPrimaryKey() string {
	for _, c := range t.columns {
		if c.pk {
			return c.name
		}
	}
	return ""
}

func (t table) getRows(limit int, offset int) ([]*row, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	query := fmt.Sprintf(`SELECT * FROM %s LIMIT %d OFFSET %d`, t.name, limit, offset)
	rows, err := t.db.executeQuery(query)
	if err != nil {
		panic(err)
	}

	return rows, err
}

func (t table) getRowByID(id int) (*row, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	query := fmt.Sprintf(`SELECT * FROM %s WHERE %s = %d`, t.name, t.getPrimaryKey(), id)
	rows, err := t.db.executeQuery(query)
	if err != nil {
		panic(err)
	}

	if len(rows) != 1 {
		return nil, fmt.Errorf("Can't get a row with id(%v) = %v in the table %v", t.getPrimaryKey(), id, t.name)
	}

	return rows[0], err
}

func (t table) addRow(values map[string]interface{}) (int64, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	columnsNames := make([]string, 0)
	putValues := make([]interface{}, 0)
	for _, column := range t.columns {
		if !column.pk {
			columnsNames = append(columnsNames, column.name)
			putValues = append(putValues, values[column.name])
		}
	}

	inputColumns := strings.Join(columnsNames, ", ")

	placeholders := "?" + strings.Repeat(", ?", len(putValues)-1)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", t.name, inputColumns, placeholders)

	id, err := t.db.execute(query, putValues...)
	if err != nil {
		return -1, err
	}

	return id.LastInsertId()
}

func (t table) updateRow(id int, values map[string]interface{}) (int64, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// create a slice like [[fieldName1 =?] [fieldName2 = ?] ...]
	setTemplate := make([]string, 0)
	putValues := make([]interface{}, 0)
	for _, column := range t.columns {
		_, ok := values[column.name]
		if !column.pk && ok {
			setTemplate = append(setTemplate, fmt.Sprintf("%v = ?", column.name))
			putValues = append(putValues, values[column.name])
		}
	}
	placeholders := strings.Join(setTemplate, ", ")

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %d", t.name, placeholders, t.getPrimaryKey(), id)

	resp, err := t.db.execute(query, putValues...)

	if err != nil {
		return -1, err
	}

	return resp.RowsAffected()
}

func (t table) deleteRowByID(id int) (int64, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", t.name, t.getPrimaryKey())

	resp, err := t.db.execute(query, id)
	if err != nil {
		return -1, err
	}

	return resp.RowsAffected()
}

// --------------------- database ---------------------
type database struct {
	conn   *sql.DB
	tables []*table
}

func (db database) String() string {
	return fmt.Sprintf("%v", db.tables)
}

func (db *database) refreshDatabaseStructure() error {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	tables := make([]*table, 0)

	qTables, err := db.conn.Query("SHOW TABLES")
	if err != nil {
		return err
	}

	for qTables.Next() {
		table := &table{
			db: db,
		}

		err = qTables.Scan(&table.name)
		if err != nil {
			return err
		}

		// Get all columns for the table
		columns := make([]*column, 0)

		qColumns, err := db.conn.Query("SHOW FULL COLUMNS FROM " + table.name)
		if err != nil {
			return err
		}

		for qColumns.Next() {
			column := &column{}
			var tmp sql.NullString
			var pk string
			var Null string
			err = qColumns.Scan(&column.name, &column.colType, &tmp, &Null, &pk, &tmp, &tmp, &tmp, &tmp)
			if err != nil {
				return err
			}

			if pk == "PRI" {
				column.pk = true
			}

			if Null == "YES" {
				column.Null = true
			}

			columns = append(columns, column)
		}
		qColumns.Close()

		// put a columns slice to a table variable
		table.columns = columns

		tables = append(tables, table)
	}
	qTables.Close()

	db.tables = tables

	return nil
}

func (db *database) getTable(tName string) (*table, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	table := &table{}
	for _, t := range db.tables {
		if tName == t.name {
			table = t
		}
	}

	if table.name == "" {
		return nil, fmt.Errorf("There is no table with the name %v", tName)
	}

	return table, nil

}

func (db *database) getAllTables() ([]string, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	result := make([]string, 0)
	for _, t := range db.tables {
		result = append(result, t.name)
	}

	return result, nil
}

func (db *database) execute(query string, values ...interface{}) (sql.Result, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	return db.conn.Exec(query, values...)
}

func (db *database) executeQuery(query string) ([]*row, error) {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// execute a SQL query
	qRows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}

	// we have an empty result. Result has got 0 rows.
	rows := make([]*row, 0)
	for qRows.Next() {

		// create input row
		qRow := &row{
			cells: make([]*cell, 0),
		}

		// prepare cells to database column types
		columnTypes, _ := qRows.ColumnTypes()
		for _, columnType := range columnTypes {
			cell := &cell{}
			cell.initiate(columnType.Name(), columnType.DatabaseTypeName())
			qRow.cells = append(qRow.cells, cell)
		}

		// read row from database
		err = qRows.Scan(qRow.getCellValues()...)
		if err != nil {
			panic(err)
		}

		// convert from database types to go types
		for _, c := range qRow.cells {
			c.convertToGoType()
		}

		//put the getted row to the resulted rows set
		rows = append(rows, qRow)
	}

	qRows.Close()

	return rows, nil
}
