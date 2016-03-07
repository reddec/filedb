package ui

import (
	"log"
	"testing"
	"time"
)

type item struct {
	Name    string
	Year    int
	Created time.Time
}

func TestSaveDB(t *testing.T) {
	id := "agent-007"
	db := DB{Root: "/tmp/db"}
	section := db.Section("russia", "moscow")
	err := section.Put(id, item{Name: "Alex", Year: 1960, Created: time.Now()})
	if err != nil {
		t.Fatal("Failed save", err)
	}
}

func TestReadDB(t *testing.T) {
	id := "agent-007"
	db := DB{Root: "/tmp/db"}
	section := db.Section("russia", "moscow")
	tp := time.Now()
	err := section.Put(id, item{Name: "Alex", Year: 1960, Created: tp})
	if err != nil {
		t.Fatal("Failed save", err)
	}
	var user item
	err = section.Get(id, &user)
	if err != nil {
		t.Fatal("Failed save", err)
	}
	if user.Name != "Alex" {
		t.Fatal("Name corrupted")
	}
	if user.Year != 1960 {
		t.Fatal("Year corrupted")
	}
	if user.Created != tp {
		t.Fatal("Time corrupted")
	}
}

func TestListDB(t *testing.T) {
	id := "agent-007"
	db := DB{Root: "/tmp/db"}
	section := db.Section("russia")
	tp := time.Now()
	err := section.Put(id, item{Name: "Alex", Year: 1960, Created: tp})
	if err != nil {
		t.Fatal("Failed save", err)
	}
	items := section.List()
	if len(items) == 0 {
		t.Fatal("Items not listed")
	}
	t.Log(items)
}

func TestChanges(t *testing.T) {
	id := "agent 007"
	db := DB{Root: "/tmp/db"}
	section := db.Section("europe", "russia", "moscow")
	if err := section.Clean(); err != nil {
		t.Fatal("Can't clean", err)
	}

	err := section.Put(id, item{Name: "Alex", Year: 1960, Created: time.Now()})
	if err != nil {
		t.Fatal("Failed save", err)
	}

	if err := section.StartNotification(); err != nil {
		t.Fatal("Start notification:", err)
	}
	defer section.StopNotification()

	go func() {
		err := section.Put(id, item{Name: "Alex", Year: 1961, Created: time.Now()})
		if err != nil {
			t.Fatal("Failed update", err)
		}
	}()
	var upd bool
	var rm bool
	var crt bool
	for rec := range section.Notification() {
		log.Println(rec)
		upd = rec.Event == Update
		break
	}
	if !upd {
		t.Fatal("Can't get update event")
	}
	go func() {
		err := section.RemoveItem(id)
		if err != nil {
			t.Fatal("Failed remove", err)
		}
	}()
	for rec := range section.Notification() {
		log.Println(rec)
		rm = rec.Event == Remove
		break
	}
	if !rm {
		t.Fatal("Can't get remove event")
	}
	go func() {
		err := section.Put(id, item{Name: "Alex", Year: 1960, Created: time.Now()})
		if err != nil {
			t.Fatal("Failed save", err)
		}

	}()
	for rec := range section.Notification() {
		log.Println(rec)
		crt = rec.Event == Create
		if err := rec.Remove(); err != nil {
			log.Println("Failed remove", err)
		}
		break

	}
	if !crt {
		t.Fatal("Can't get create event")
	}
}
