package main

import (
	"context"
	"net/url"
	"time"
	"regexp"

	"git.zabbix.com/ap/plugin-support/uri"
	"git.zabbix.com/ap/plugin-support/zbxerr"
	"git.zabbix.com/ap/plugin-support/plugin"
)

const (
	pluginName = "zoracle"
	hkInterval = 10
	sqlExt     = ".sql"
)

// Plugin inherits plugin.Base and store plugin-specific data.
type Plugin struct {
	plugin.Base
	connMgr *ConnManager
	options PluginOptions
}

// impl is the pointer to the plugin implementation.
var impl Plugin

// Export implements the Exporter interface.
func (p *Plugin) Export(key string, rawParams []string, _ plugin.ContextProvider) (result interface{}, err error) {
	
    p.Tracef("[Export] begin for key : %s", key)

	//Sessions map[string]Session
	params, extraParams, err := metrics[key].EvalParams(rawParams, p.options.Sessions)

	p.Tracef("[Export] params : %s", params)
	p.Tracef("[Export] extraParams : %s", extraParams)
	if err != nil {
		return nil, err
	}

	service := url.QueryEscape(params["Service"])
	uri, err := uri.NewWithCreds(params["URI"]+"?service="+service, params["User"], params["Password"], uriDefaults)

	if err != nil {
		return nil, err
	}

	handleMetric := getHandlerFunc(key)
	
	if handleMetric == nil {
		return nil, zbxerr.ErrorUnsupportedMetric
	}

	p.Tracef("[Export] grab connection")
	conn, err := p.connMgr.GetConnection(p, *uri)
	if err != nil {
		p.Tracef("[Export] error grabbing connection")
		// Special logic of processing connection errors should be used if oracle.ping is requested
		// because it must return pingFailed if any error occurred.
	
		if key == keyPing {
			p.Tracef("[Export] check if error containt ORA-XXXXX")
			var rgx = regexp.MustCompile(`ORA-[0-9]{5}.*`)
			rs := rgx.FindStringSubmatch(err.Error())

			if len(rs) > 0 {
				p.Tracef("[Export] found error ORA-XXXXX")
				p.Tracef("[Export] returning -> %s", rs[0])
				return rs[0], nil
			}

			p.Tracef("[Export] didn't found ORA-XXXXX so returning pingfailed")
			return pingFailed, nil
		}

        p.Tracef("[Export] returning error when key != KeyPing")		
		p.Errf(err.Error())
		return nil, err
	}

	ctx, cancel := context.WithTimeout(conn.ctx, conn.callTimeout)

	defer cancel()

	p.Tracef("[Export] executing handleMetric for key : %s", key)
	result, err = handleMetric(p, ctx, conn, params, extraParams...)
	p.Tracef("[Export] after executing handleMetric for key : %s", key)

	if err != nil {
		p.Errf(err.Error())
		p.Tracef("[Export] finished with error!!! key : %s", key)
		p.Tracef("[Export] error: %v", err)
	} else {
		p.Tracef("[Export] finished for key : %s", key)
	}

	p.Tracef("[Export] returning")
	return result, err
}

// Start implements the Runner interface and performs initialization when plugin is activated.
func (p *Plugin) Start() {
	p.connMgr = NewConnManager(
		time.Duration(p.options.KeepAlive)*time.Second,
		time.Duration(p.options.ConnectTimeout)*time.Second,
		time.Duration(p.options.CallTimeout)*time.Second,
		hkInterval*time.Second,
	)
}

// Stop implements the Runner interface and frees resources when plugin is deactivated.
func (p *Plugin) Stop() {
	p.connMgr.Destroy()
	p.connMgr = nil
}
