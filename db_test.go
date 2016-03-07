package ui

import (
	"fmt"
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
			t.Fatal("Failed remove", err)
		}
		break

	}
	if !crt {
		t.Fatal("Can't get create event")
	}
}

// Examples

func ExampleCRUD() {
	// Create new database located to /tmp/example-db directory
	db := DB{Root: "/tmp/example-db"}
	// Create new item with ID = 001 into root section
	err := db.Put("Hello world", "001")
	if err != nil {
		panic(err)
	}
	// Complex item also supported
	// it will be serialized by JSON encoder (may be changed in future)
	var user struct {
		Name string
		Year int
	}
	user.Name = "Alex"
	user.Year = 1900
	err = db.Put(user, "002")
	if err != nil {
		panic(err)
	}

	// Get item by ID in root section
	var item string
	err = db.Get(&item, "001")
	if err != nil {
		panic(err)
	}
	fmt.Println(item)
	// ... or get list of all items and subsections (only id's)
	// db.List()

	// Remove item by ID in root section
	err = db.RemoveItem("001")
	if err != nil {
		panic(err)
	}

	// Output: Hello world
}

func ExampleSubsection() {
	// Create new database located to /tmp/example-db directory
	db := DB{Root: "/tmp/example-db"}
	// For example, we have to save users separated by country
	// We can do it directly
	// Here: ID = 001, section = europe/germany
	err := db.Put("Merkel", "001", "europe", "germany")
	if err != nil {
		panic(err)
	}
	// Also we can use subsection
	germany := db.Section("europe", "germany")
	// Now add more people to Germany
	if err := germany.Put("002", "Bismark"); err != nil {
		panic(err)
	}
	if err := germany.Put("003", "Kant"); err != nil {
		panic(err)
	}
	// And get list of people in Germany
	for _, record := range germany.List() {
		// Get people
		var user string
		err := record.Get(&user)
		if err != nil {
			panic(err)
		}
		fmt.Print(user, " ")
	}
	// Output: Merkel Bismark Kant
}
