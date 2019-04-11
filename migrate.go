package coresql

import (
	"flag"
	"log"
)

// Migrate represents functionality for
type Migrate interface {
	Up() error
	Down() error
}

// HandleMigrationArguments should be invoked to handle command line arguments for running migrations.
func HandleMigrationArguments(migrate Migrate) {
	switch flag.Arg(1) {
	case "migrate":
		switch flag.Arg(2) {
		case "up":
			if err := migrate.Up(); err != nil {
				log.Fatalln(err)
			}
		case "down":
			if err := migrate.Down(); err != nil {
				log.Fatalln(err)
			}
		default:
			log.Fatalln("need to provide a command to 'migrate'")
		}
	}
}

// MustMigrateUp will attempt to migrate your database up.
func MustMigrateUp(migrate Migrate) {
	if err := migrate.Up(); err != nil {
		log.Fatalln(err)
	}
}