package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"git.zabbix.com/ap/plugin-support/zbxerr"
)

// customQueryHandler executes custom user queries
func customQueryHandler(
	//mn
	p *Plugin, 
	ctx context.Context, conn OraClient,
	params map[string]string, extraParams ...string) (interface{}, error) {
	p.Tracef("[customQueryHandler] begin")

	query := params["Query"]
	queryArgs := make([]interface{}, len(extraParams))
	for i, v := range extraParams {
		queryArgs[i] = v
	}

	p.Tracef("[customQueryHandler] before execute query")
	rows, err := conn.Query(ctx, query, queryArgs...)
	if err != nil {
		p.Tracef("[customQueryHandler] error executing query")
		p.Tracef("[customQueryHandler] error: %v", err)
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}
	defer rows.Close()

	// JSON marshaling
	var data []string

	p.Tracef("[customQueryHandler] get columns")
	columns, err := rows.Columns()
	if err != nil {
		p.Tracef("[customQueryHandler] error get columns")
		p.Tracef("[customQueryHandler] error: %v", err)
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}

	values := make([]interface{}, len(columns))
	valuePointers := make([]interface{}, len(values))

	for i := range values {
		valuePointers[i] = &values[i]
	}

	results := make(map[string]interface{})

	p.Tracef("[customQueryHandler] begin read recordset")
	for rows.Next() {
		p.Tracef("[customQueryHandler] read recordset line")
		err = rows.Scan(valuePointers...)
		if err != nil {
			p.Tracef("[customQueryHandler] Error Scan")
			p.Tracef("[customQueryHandler] Error: %v", err)

			if err == sql.ErrNoRows {
				p.Tracef("[customQueryHandler] no rows")
				return nil, zbxerr.ErrorEmptyResult.Wrap(err)
			}

			return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
		}

		p.Tracef("[customQueryHandler] fill results")
		for i, value := range values {
			results[columns[i]] = value
		}

		p.Tracef("[customQueryHandler] convert results to json")
		jsonRes, _ := json.Marshal(results)
		p.Tracef("[customQueryHandler] append to data")
		data = append(data, strings.TrimSpace(string(jsonRes)))
		p.Tracef("[customQueryHandler] done appending")
	}
	p.Tracef("[customQueryHandler] end read recordset")

	return "[" + strings.Join(data, ",") + "]", nil
}
