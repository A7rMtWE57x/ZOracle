
package main

import (
	"context"

	"git.zabbix.com/ap/plugin-support/metric"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/uri"
)

const (
	keyCustomQuery            = "zoracle.custom.query"
	keyPing                   = "zoracle.ping"
)

// handlerFunc defines an interface must be implemented by handlers.
type handlerFunc func(ctx context.Context, conn OraClient,
	params map[string]string, extraParams ...string) (res interface{}, err error)

// getHandlerFunc returns a handlerFunc related to a given key.
func getHandlerFunc(key string) handlerFunc {
	switch key {
	case keyCustomQuery:
		return customQueryHandler
	case keyPing:
		return pingHandler
	default:
		return nil
	}
}

var uriDefaults = &uri.Defaults{Scheme: "tcp", Port: "1521"}

// Common params: [URI|Session][,User][,Password][,Service]
var (
	paramURI = metric.NewConnParam("URI", "URI to connect or session name.").
			WithDefault(uriDefaults.Scheme + "://localhost:" + uriDefaults.Port).WithSession().
			WithValidator(uri.URIValidator{Defaults: uriDefaults, AllowedSchemes: []string{"tcp"}})
	paramUsername = metric.NewConnParam("User", "Oracle user.").WithDefault("")
	paramPassword = metric.NewConnParam("Password", "User's password.").WithDefault("")
	paramService  = metric.NewConnParam("Service", "Service name to be used for connection.").
			WithDefault("XE")
)

var metrics = metric.MetricSet{
	keyCustomQuery: metric.New("Returns result of a custom query.",
		[]*metric.Param{paramURI, paramUsername, paramPassword, paramService,
			metric.NewParam("Query", "SQL string with custom query ").SetRequired(),
		}, true),

	keyPing: metric.New("Tests if connection is alive or not.",
		[]*metric.Param{paramURI, paramUsername, paramPassword, paramService}, false),
}

func init() {
	plugin.RegisterMetrics(&impl, pluginName, metrics.List()...)
}
