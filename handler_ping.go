package main

import (
	"context"
	"fmt"
)

const (
	pingFailed = 0
	pingOk     = 1
)

// pingHandler queries 'SELECT 1 FROM DUAL' and returns pingOk if a connection is alive or pingFailed otherwise.
func pingHandler(ctx context.Context, conn OraClient, params map[string]string, _ ...string) (interface{}, error) {
	var res int

	row, err := conn.QueryRow(ctx, fmt.Sprintf("SELECT %d FROM DUAL", pingOk))
	if err != nil {
		return pingFailed, nil
	}

	err = row.Scan(&res)

	if err != nil || res != pingOk {
		return pingFailed, nil
	}

	return pingOk, nil
}
