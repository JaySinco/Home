package core

import (
	"database/sql"
	"encoding/base64"
	"sync"

	_ "github.com/lib/pq"
)

func Dtbs() *sql.DB {
	globalDtbs.o.Do(func() {
		var err error
		globalDtbs.data, err = dtbsSetup(Config().Core.Driver, Config().Core.DtbsToken)
		if err != nil {
			Fatal("failed to set up database: %s", err)
		}
	})
	return globalDtbs.data
}

var globalDtbs struct {
	data *sql.DB
	o    sync.Once
}

func dtbsSetup(driver string, token string) (*sql.DB, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	Info("load %s", data)
	db, err := sql.Open(driver, string(data))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
