package main

import "testing"
import "gopkg.in/mgo.v2"
import "gopkg.in/mgo.v2/bson"
import "time"

func setupDB(t *testing.T) *trackDB {
	db := trackDB{
		"mongodb://localhost",
		"testtrackdb",
		"tracks",
	}

	session, err := mgo.Dial(db.HostURL)
	defer session.Close()
	if err != nil {
		t.Error(err)
	}

	return &db
}

func tearDownDB(t *testing.T, db *trackDB) {
	session, err := mgo.Dial(db.HostURL)
	defer session.Close()
	if err != nil {
		t.Error(err)
	}

	session.DB(db.DatabaseName).DropDatabase()
	if err != nil {
		t.Error(err)
	}
}

func TestTrackDB_Add(t *testing.T) {
	db := setupDB(t)
	defer tearDownDB(t, db)

	db.Init()
	if db.Count() != 0 {
		t.Error("database not properly initialized. track Count() should be 0")
	}

	track := Track{"test", time.Now(), "Gerd", "Glider1", "123Glider", 5, "some line", bson.NewObjectIdWithTime(time.Now()),}
	db.Add(track)

	if db.Count() != 1 {
		t.Error("adding new student failed!")
	}
}

func TestTrackDB_Get(t *testing.T) {
	db := setupDB(t)
	defer tearDownDB(t, db)

	db.Init()
	if db.Count() != 0 {
		t.Error("database not properly initialized. track Count() should be 0")
	}

	track := Track{"test", time.Now(), "Gerd", "Glider1", "123Glider", 5, "some line", 	bson.NewObjectIdWithTime(time.Now())}
	db.Add(track)

	if db.Count() != 1 {
		t.Error("adding new student failed!")
	}

	newTrack, ok := db.Get(track.ID)
	if !ok {
		t.Error("Could not get track")
	}

	if newTrack.Pilot != track.Pilot ||
		newTrack.Glider != track.Glider ||
		newTrack.GliderID != track.GliderID ||
		newTrack.HDate != track.HDate ||
		newTrack.URL != track.URL ||
		newTrack.ID != track.ID {
		t.Error("tracks do not match")
	}
}
