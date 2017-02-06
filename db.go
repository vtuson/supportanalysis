package main

import (
	"gopkg.in/mgo.v2"
	"log"
)

type Database struct {
	dburi    string
	Session  *mgo.Session
	Database *mgo.Database
	Name     string
}

func NewDb(name string) *Database {
	db := Database{Name: name}
	return &db
}

//connect to a specific uri for a mongodb database
func (db *Database) Connect(uri string) error {
	//Open session to uri
	sess, err := mgo.Dial(uri)
	if err != nil {
		return err
	}
	db.Session = sess
	sess.SetSafe(&mgo.Safe{})
	db.Database = sess.DB(db.Name)
	db.dburi = uri
	return nil
}

func (db *Database) SetLogin(l *log.Logger) {
	mgo.SetLogger(l)
	mgo.SetDebug(true)
}
func (db *Database) SetLoginOff() {
	mgo.SetDebug(false)
}

//copy from provided database
func CopyDatabase(db *Database) *Database {
	//Open session to uri
	var newDb Database
	sess := db.Session.Copy()
	if sess == nil {
		log.Println("could not create session")
		return nil
	}
	newDb.Session = sess
	newDb.Database = newDb.Session.DB(db.Name)
	return &newDb
}

//closses session in the database
func (db *Database) Close() {
	db.Session.Close()
}