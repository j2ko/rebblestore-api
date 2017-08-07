package main

import (
	"database/sql"
	"os"
	"testing"

	"github.com/adams-sarah/test2doc/test"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var server *test.Server

func TestMain(m *testing.M) {
	var err error

	db, err := sql.Open("sqlite3", "./foo_test.db")
	if err != nil {
		panic("Could not connect to database" + err.Error())
	}

	context := &handlerContext{db}

	var r = Handlers(context)
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

	os.Exit(exitCode)
}
