package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sync"
	"time"

	"git.zabbix.com/ap/plugin-support/uri"

	"git.zabbix.com/ap/plugin-support/log"
	"git.zabbix.com/ap/plugin-support/zbxerr"
	"github.com/godror/godror"
	"github.com/omeid/go-yarn"
)

type OraClient interface {
	Query(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error)
	QueryRow(ctx context.Context, query string, args ...interface{}) (row *sql.Row, err error)
	WhoAmI() string
}

type OraConn struct {
	client         *sql.DB
	callTimeout    time.Duration
	version        godror.VersionInfo
	lastTimeAccess time.Time
	ctx            context.Context
	username       string
}

var errorQueryNotFound = "query %q not found"

// Query wraps DB.QueryContext.
func (conn *OraConn) Query(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	rows, err = conn.client.QueryContext(ctx, query, args...)

	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}

	return
}

// Query wraps DB.QueryRowContext.
func (conn *OraConn) QueryRow(ctx context.Context, query string, args ...interface{}) (row *sql.Row, err error) {
	row = conn.client.QueryRowContext(ctx, query, args...)

	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}

	return
}

// WhoAmI returns a current username.
func (conn *OraConn) WhoAmI() string {
	return conn.username
}

// updateAccessTime updates the last time a connection was accessed.
func (conn *OraConn) updateAccessTime() {
	conn.lastTimeAccess = time.Now()
}

// ConnManager is thread-safe structure for manage connections.
type ConnManager struct {
	sync.Mutex
	connMutex      sync.Mutex
	connections    map[uri.URI]*OraConn
	keepAlive      time.Duration
	connectTimeout time.Duration
	callTimeout    time.Duration
	Destroy        context.CancelFunc
	queryStorage   yarn.Yarn
}

// NewConnManager initializes connManager structure and runs Go Routine that watches for unused connections.
func NewConnManager(keepAlive, connectTimeout, callTimeout,
	hkInterval time.Duration) *ConnManager {
	ctx, cancel := context.WithCancel(context.Background())

	connMgr := &ConnManager{
		connections:    make(map[uri.URI]*OraConn),
		keepAlive:      keepAlive,
		connectTimeout: connectTimeout,
		callTimeout:    callTimeout,
		Destroy:        cancel, // Destroy stops originated goroutines and closes connections.
	}

	go connMgr.housekeeper(ctx, hkInterval)

	return connMgr
}

// closeUnused closes each connection that has not been accessed at least within the keepalive interval.
func (c *ConnManager) closeUnused() {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	for uri, conn := range c.connections {
		if time.Since(conn.lastTimeAccess) > c.keepAlive {
			conn.client.Close()
			delete(c.connections, uri)
			log.Debugf("[%s] Closed unused connection: %s", pluginName, uri.Addr())
		}
	}
}

// closeAll closes all existed connections.
func (c *ConnManager) closeAll() {
	c.connMutex.Lock()
	for uri, conn := range c.connections {
		conn.client.Close()
		delete(c.connections, uri)
	}
	c.connMutex.Unlock()
}

// housekeeper repeatedly checks for unused connections and closes them.
func (c *ConnManager) housekeeper(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			c.closeAll()

			return
		case <-ticker.C:
			c.closeUnused()
		}
	}
}

// create creates a new connection with given credentials.
func (c *ConnManager) create(uri uri.URI) (*OraConn, error) {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	if _, ok := c.connections[uri]; ok {
		// Should never happen.
		panic("connection already exists")
	}

	ctx := godror.ContextWithTraceTag(
		context.Background(),
		godror.TraceTag{
			ClientInfo: "zbx_monitor",
			Module:     godror.DriverName,
		})

	service, err := url.QueryUnescape(uri.GetParam("service"))
	if err != nil {
		return nil, err
	}

	connectString := fmt.Sprintf(`(DESCRIPTION=(ADDRESS=(PROTOCOL=tcp)(HOST=%s)(PORT=%s))`+
		`(CONNECT_DATA=(SERVICE_NAME="%s"))(CONNECT_TIMEOUT=%d)(RETRY_COUNT=0))`,
		uri.Host(), uri.Port(), service, c.connectTimeout/time.Second)

	connector := godror.NewConnector(godror.ConnectionParams{
		StandaloneConnection: true,
		CommonParams: godror.CommonParams{
			Username:      uri.User(),
			ConnectString: connectString,
			Password:      godror.NewPassword(uri.Password()),
		},
	})

	client := sql.OpenDB(connector)

	serverVersion, err := godror.ServerVersion(ctx, client)
	if err != nil {
		return nil, err
	}

	c.connections[uri] = &OraConn{
		client:         client,
		callTimeout:    c.callTimeout,
		version:        serverVersion,
		lastTimeAccess: time.Now(),
		ctx:            ctx,
		username:       uri.User(),
	}

	log.Debugf("[%s] Created new connection: %s", pluginName, uri.Addr())

	return c.connections[uri], nil
}

// get returns a connection with given uri if it exists and also updates lastTimeAccess, otherwise returns nil.
func (c *ConnManager) get(uri uri.URI) *OraConn {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	if conn, ok := c.connections[uri]; ok {
		conn.updateAccessTime()
		return conn
	}

	return nil
}

// GetConnection returns an existing connection or creates a new one.
func (c *ConnManager) GetConnection(uri uri.URI) (conn *OraConn, err error) {
	c.Lock()
	defer c.Unlock()

	conn = c.get(uri)

	if conn == nil {
		conn, err = c.create(uri)
	}

	if err != nil {
		if oraErr, isOraErr := godror.AsOraErr(err); isOraErr {
			err = zbxerr.ErrorConnectionFailed.Wrap(oraErr)
		} else {
			err = zbxerr.ErrorConnectionFailed.Wrap(err)
		}
	}

	return
}