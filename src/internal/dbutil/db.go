package dbutil

import (
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

const (
	// DefaultMaxOpenConns is the argument passed to SetMaxOpenConns
	DefaultMaxOpenConns = 3
)

type dBConfig struct {
	host           string
	port           int
	user, password string
	name           string
	maxOpenConns   int
}

// NewDB creates a new DB.
func NewDB(opts ...Option) (*sqlx.DB, error) {
	dbc := &dBConfig{
		user:         "postgres",
		maxOpenConns: DefaultMaxOpenConns,
	}
	for _, opt := range opts {
		opt(dbc)
	}
	fields := map[string]string{
		"sslmode": "disable",
	}
	if dbc.host != "" {
		fields["host"] = dbc.host
	}
	if dbc.port != 0 {
		fields["port"] = strconv.Itoa(dbc.port)
	}
	if dbc.name != "" {
		fields["dbname"] = dbc.name
	}
	if dbc.user != "" {
		fields["user"] = dbc.user
	}
	if dbc.password != "" {
		fields["password"] = dbc.password
	}
	var dsnParts []string
	for k, v := range fields {
		dsnParts = append(dsnParts, k+"="+v)
	}
	dsn := strings.Join(dsnParts, " ")
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if dbc.maxOpenConns != 0 {
		db.SetMaxOpenConns(dbc.maxOpenConns)
	}
	return db, nil
}
