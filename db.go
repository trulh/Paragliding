package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type trackDB struct {
	HostURL             string
	DatabaseName        string
	TrackCollectionName string
}

type webhookDB struct {
	HostURL               string
	DatabaseName          string
	WebhookCollectionName string
}

func (db *trackDB) Init() {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

}

func (db *webhookDB) Init() {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

}

func (db *trackDB) Add(s Track) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	err = session.DB(db.DatabaseName).C(db.TrackCollectionName).Insert(s)
	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
	}
}

func (db *webhookDB) Add(s Webhook) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	err = session.DB(db.DatabaseName).C(db.WebhookCollectionName).Insert(s)
	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
	}
}

func (db *trackDB) Count() int {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	count, err := session.DB(db.DatabaseName).C(db.TrackCollectionName).Count()
	if err != nil {
		fmt.Printf("error in Count(): %v", err.Error())
		return -1
	}
	return count

}

func (db *webhookDB) Count() int {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	count, err := session.DB(db.DatabaseName).C(db.WebhookCollectionName).Count()
	if err != nil {
		fmt.Printf("error in Count(): %v", err.Error())
		return -1
	}
	return count

}

func (db *trackDB) Get(keyID string) (Track, bool) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	track := Track{}
	err = session.DB(db.DatabaseName).C(db.TrackCollectionName).Find(bson.M{"id": keyID}).One(&track)
	if err != nil {
		return track, false
	}

	return track, true
}

func (db *webhookDB) Get(keyID string) (Webhook, bool) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()

	Hook := Webhook{}
	err = session.DB(db.DatabaseName).C(db.WebhookCollectionName).Find(bson.M{"id": keyID}).One(&Hook)
	if err != nil {
		return Hook, false
	}

	return Hook, true
}

func (db *trackDB) Delete() (int, bool) {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()
	tracks := session.DB(db.DatabaseName).C(db.TrackCollectionName)

	count, err := tracks.Count()
	if err != nil {
		return 0, false
	}

	tracks.RemoveAll(nil)
	return count, true
}

func (db *webhookDB) Delete(keyID string) bool {
	session, err := mgo.Dial(db.HostURL)
	if err != nil {
		panic(err)
	}

	defer session.Close()
	err = session.DB(db.DatabaseName).C(db.WebhookCollectionName).Remove(bson.M{"id": keyID})
	if err != nil {
		return false
	}

	return true
}
