package main

import (
	"database/sql"
	"os"
	"pebble-dev/rebblestore-api/db"
	"pebble-dev/rebblestore-api/rebbleHandlers"
	"testing"

	"github.com/adams-sarah/test2doc/test"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var server *test.Server

func TestMain(m *testing.M) {
	var err error

	database, err := sql.Open("sqlite3", "./RebbleAppStore.db")
	if err != nil {
		panic("Could not connect to database" + err.Error())
	}

	dbHandler := db.Handler{database}
	context := &rebbleHandlers.HandlerContext{&dbHandler}

	var r = rebbleHandlers.Handlers(context)
	r.KeepContext = true
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	test.RegisterURLVarExtractor(mux.Vars)

	//server, err = test.NewServer(r)
	server, err = test.NewServer(loggedRouter)
	if err != nil {
		panic(err.Error())
	}
	defer server.Close()
	exitCode := m.Run()

	server.Finish()

	os.Remove("./foo_test.db")
	os.Exit(exitCode)
}
