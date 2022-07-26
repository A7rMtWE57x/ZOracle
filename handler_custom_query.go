package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"git.zabbix.com/ap/plugin-support/zbxerr"
)

// customQueryHandler executes custom user queries
func customQueryHandler(ctx context.Context, conn OraClient,
	params map[string]string, extraParams ...string) (interface{}, error) {
	query := params["Query"]

	queryArgs := make([]interface{}, len(extraParams))
	for i, v := range extraParams {
		queryArgs[i] = v
	}

	rows, err := conn.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}
	defer rows.Close()

	// JSON marshaling
	var data []string

	columns, err := rows.Columns()
	if err != nil {
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}

	values := make([]interface{}, len(columns))
	valuePointers := make([]interface{}, len(values))

	for i := range values {
		valuePointers[i] = &values[i]
	}

	results := make(map[string]interface{})

	for rows.Next() {
		err = rows.Scan(valuePointers...)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, zbxerr.ErrorEmptyResult.Wrap(err)
			}

			return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
		}

		for i, value := range values {
			results[columns[i]] = value
		}

		jsonRes, _ := json.Marshal(results)
		data = append(data, strings.TrimSpace(string(jsonRes)))
	}

	return "[" + strings.Join(data, ",") + "]", nil
}
