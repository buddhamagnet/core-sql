package coresql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"

	// Since we are most likely going to be only retrieving migrations from file source,
	// it's prudent that we include this side effect inside of this package and not
	// having to import it inside each and every project.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	waitMaxTries = 60
	waitTimeout  = 5 * time.Second
	waitCooldown = 10 * time.Millisecond
)

var errTimeout = fmt.Errorf("could not connect to database: timed out")

// DB represents a wrapper for SQL DB providing extra methods.
type DB struct {
	*sql.DB
}

// Check will attempt to ping the database to see if the connection is still alive.
func (db *DB) Check() ([]string, bool) {
	if err := db.Ping(); err != nil {
		return []string{err.Error()}, false
	}
	return []string{"database connection ok"}, true
}

// MustWait will call the Wait method and crash if it cant be performed.
func (db *DB) MustWait() {
	if err := db.Wait(); err != nil {
		log.Fatal(err)
	}
}

// Wait will attempt to connect to a database and block until it connects.
func (db *DB) Wait() error {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()

	tries := 0
	doneC := make(chan struct{}, 1)
	sem := make(chan struct{}, 1)

	ping := func(ctx context.Context) {
		err := db.PingContext(ctx)
		if err == nil {
			doneC <- struct{}{}
			return
		}
		time.Sleep(waitCooldown)
		tries++
		<-sem
	}

	for {
		select {
		case sem <- struct{}{}:
			if tries >= waitMaxTries {
				return fmt.Errorf("could not connect to database: attempt limit (%d) exceeded", waitMaxTries)
			}
			go ping(ctx)
		case <-ctx.Done():
			return fmt.Errorf("could not connect to database: timeout")
		case <-doneC:
			return nil
		}
	}
}

// Open will attempt to open a new database connection.
func Open(driverName, dsn string) (*DB, error) {
	switch driverName {
	case "mysql":
	default:
		dsn = fmt.Sprintf("%s://%s", driverName, dsn)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbC := make(chan *sql.DB, 1)
	errC := make(chan error, 1)

	go func(driverName, dsn string) {
		db, err := sql.Open(driverName, dsn)
		if err != nil {
			errC <- err
			return
		}
		dbC <- db
	}(driverName, dsn)

	select {
	case db := <-dbC:
		// see: https://github.com/go-sql-driver/mysql/issues/674
		db.SetMaxIdleConns(0)
		return &DB{db}, nil
	case err := <-errC:
		return nil, err
	case <-ctx.Done():
		return nil, errTimeout
	}
}

// MustOpen will crash your program unless a database could be retrieved.
func MustOpen(driverName, dsn string) *DB {
	db, err := Open(driverName, dsn)
	if err != nil {
		log.Fatalln(err)
	}
	return db
}

// OpenMigration will open a migration instance.
func OpenMigration(driverName, dsn, sourceURL string) (*migrate.Migrate, error) {
	log.Printf("OpenMigration: driver %s dsn %s sourceURL %s\n", driverName, dsn, sourceURL)

	dsn = fmt.Sprintf("%s://%s", driverName, dsn)

	log.Printf("OpenMigration: dsn %s\n", dsn)

	migration, err := migrate.New(sourceURL, dsn)

	log.Printf("OpenMigration: migration %#v\n", migration)
	if err != nil {
		return nil, err
	}
	return migration, nil
}

// MustOpenMigration will open a migration instance with and crashes if unsuccessful.
func MustOpenMigration(driverName, dsn, sourceURL string) *migrate.Migrate {
	migration, err := OpenMigration(driverName, dsn, sourceURL)
	if err != nil {
		log.Fatalln(err)
	}
	return migration
}

// OpenWithMigrations opens a database connection with an associated migration instance.
func OpenWithMigrations(driverName, dsn, sourceURL string) (*DB, *migrate.Migrate, error) {
	log.Printf("OpenWithMigrations: driver %s dsn %s sourceURL %s\n", driverName, dsn, sourceURL)
	migration, err := OpenMigration(driverName, dsn, sourceURL)
	if err != nil {
		return nil, nil, err
	}
	database, err := Open(driverName, dsn)
	if err != nil {
		return nil, nil, err
	}
	return database, migration, nil
}

// MustOpenWithMigrations opens a database connection with an associated migration instance and crashes if unsuccessful.
func MustOpenWithMigrations(driverName, dsn, sourceURL string) (*DB, *migrate.Migrate) {
	database, migrations, err := OpenWithMigrations(driverName, dsn, sourceURL)
	log.Printf("MustOpenWithMigrations: driver %s dsn %s sourceURL %s\n", driverName, dsn, sourceURL)
	if err != nil {
		log.Fatalln(err)
	}
	return database, migrations
}
