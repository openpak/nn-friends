package database

import (
	_ "github.com/lib/pq"

	"github.com/PretendoNetwork/friends/globals"
	sqlmanager "github.com/PretendoNetwork/sql-manager"
)

var Manager *sqlmanager.SQLManager

func ConnectPostgres() {
	var err error

	Manager, err = sqlmanager.NewSQLManager("postgres", globals.Config.PostgresURI, globals.Config.PostgresMaxConnections)
	if err != nil {
		globals.Logger.Critical(err.Error())
	}

	globals.Logger.Success("Connected to Postgres!")

	initPostgresWiiU()
	initPostgres3DS()
}
