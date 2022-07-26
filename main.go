
package main

import (
	"fmt"

	"git.zabbix.com/ap/plugin-support/plugin/container"
)

func main() {
	h, err := container.NewHandler(impl.Name())
	if err != nil {
		panic(fmt.Sprintf("failed to create plugin handler %s", err.Error()))
	}

	impl.Logger = &h

	err = h.Execute()
	if err != nil {
		panic(fmt.Sprintf("failed to execute plugin handler %s", err.Error()))
	}
}
