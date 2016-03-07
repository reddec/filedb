package ui

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

const formatExtension = ".json"

// Record is not pointed to item
var ErrNotItem = errors.New("Not an item")

// Event type of record's changes
type Event int

// Record change events
const (
	Update Event = iota // Record content changed
	Remove              //Record removed
	Create              //Record created or copied
)

var _eventNames = []string{"Update", "Remove", "Create"}

func (ev Event) String() string {
	if int(ev) >= 0 && int(ev) < len(_eventNames) {
		return _eventNames[ev]
	}
	return "Unknown event"
}

// DB is file based database.
// Sections - level of folders.
// Items - JSON files.
// Names automatically url-encoded.
type DB struct {
	Root  string
	guard sync.RWMutex
}

func (db *DB) lockReadDB() {
	//TODO: file lock
	db.guard.RLock()
}

func (db *DB) unlockReadDB() {
	//TODO: file unlock
	db.guard.RUnlock()
}

func (db *DB) lockWriteDB() {
	//TODO: file lock
	db.guard.Lock()
}

func (db *DB) unlockWriteDB() {
	//TODO: file unlock
	db.guard.Unlock()
}

// DirLocation - get real directory (relative to Root) location of specified sections
func (db *DB) DirLocation(sectionPath ...string) string {
	escaped := make([]string, len(sectionPath)+1)
	escaped[0] = db.Root
	for i := 0; i < len(sectionPath); i++ {
		escaped[1+i] = url.QueryEscape(sectionPath[i])
	}
	return path.Join(escaped...)
}

// FileLocation - get real file (relative to Root) location of specified sections with format extension
// This function doesn't check file
func (db *DB) FileLocation(sectionPath ...string) string {
	return db.DirLocation(sectionPath...) + formatExtension
}

// Section - encapsulate section to single object and provide notifications
func (db *DB) Section(sectionPath ...string) *Section {
	return &Section{db: db, sections: sectionPath}
}

// Put or update item
func (db *DB) Put(object interface{}, id string, sectionPath ...string) error {
	db.lockWriteDB()
	defer db.unlockWriteDB()
	location := db.FileLocation(append(sectionPath, id)...)
	if err := os.MkdirAll(path.Dir(location), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(object, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(location, data, 0755)
}

// RemoveItem - removes saved item from filesystem
func (db *DB) RemoveItem(id string, sectionPath ...string) error {
	db.lockWriteDB()
	defer db.unlockWriteDB()
	return os.Remove(db.FileLocation(append(sectionPath, id)...))
}

// Clean sectoions - remove all subsections and saved items
func (db *DB) Clean(sectionPath ...string) error {
	db.lockWriteDB()
	defer db.unlockWriteDB()
	return os.RemoveAll(db.DirLocation(sectionPath...))
}

// Get signle item from filesystem and unmarshall it by JSON decoder.
// Decoder may be changed in future releases
func (db *DB) Get(destination interface{}, id string, sectionPath ...string) error {
	db.lockReadDB()
	defer db.unlockReadDB()
	location := db.FileLocation(append(sectionPath, id)...)
	data, err := ioutil.ReadFile(location)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, destination)
}

// List all saved records (items and subsections) in specified section
func (db *DB) List(sectionPath ...string) []Record {
	db.lockReadDB()
	defer db.unlockReadDB()
	var ans []Record
	locations := db.DirLocation(sectionPath...)
	infos, err := ioutil.ReadDir(locations)
	if err != nil {
		return ans
	}
	for _, info := range infos {
		rec := Record{}
		name, _ := url.QueryUnescape(info.Name())
		rec.db = db
		rec.Subsection = info.IsDir()
		rec.Name, _ = url.QueryUnescape(name)
		rec.Section = sectionPath
		if !rec.Subsection {
			idx := strings.LastIndex(rec.Name, ".")
			if idx >= 0 {
				rec.Name = rec.Name[:idx]
			}
		}
		ans = append(ans, rec)
	}
	return ans
}

// Section in database. Wrapper for Get/Put/Remove/List operations and provides
// notifications
type Section struct {
	db       *DB
	sections []string
	watcher  *fsnotify.Watcher
	notify   chan RecordEvent
}

// Notification channel. Will be created after StartNotification()
func (s *Section) Notification() <-chan RecordEvent {
	return s.notify
}

// Location in real filesystem
func (s *Section) Location() string {
	return s.db.DirLocation(s.sections...)
}

// Clean section
func (s *Section) Clean() error {
	return s.db.Clean(s.sections...)
}

// RemoveItem - remove one item in current section
func (s *Section) RemoveItem(id string) error {
	return s.db.RemoveItem(id, s.sections...)
}

// StartNotification - create notification channel and starts listen
// to any file changes in current section (not in subsections).
func (s *Section) StartNotification() error {
	var err error
	err = os.MkdirAll(s.Location(), 0755)
	if err != nil {
		return err
	}
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = s.watcher.Add(s.Location())
	if err != nil {
		return err
	}
	s.notify = make(chan RecordEvent)
	go func() {
		defer close(s.notify)
		for event := range s.watcher.Events {

			relative, err := filepath.Rel(s.db.Root, event.Name)
			if err != nil {
				continue
			}
			rec := RecordEvent{}
			rec.db = s.db
			rec.Subsection = !strings.HasSuffix(event.Name, formatExtension)
			rec.Name, _ = url.QueryUnescape(path.Base(relative))
			rec.Section = strings.Split(path.Dir(relative), string(os.PathSeparator))
			if event.Op&fsnotify.Write == fsnotify.Write {
				rec.Event = Update
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				rec.Event = Create
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				rec.Event = Remove
			} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			} else {
				rec.Event = Update
			}
			if !rec.Subsection {
				idx := strings.LastIndex(rec.Name, ".")
				if idx >= 0 {
					rec.Name = rec.Name[:idx]
				}
			}
			s.notify <- rec
		}
	}()
	return nil
}

// StopNotification - stops notification (if it created)
func (s *Section) StopNotification() {
	if s.watcher != nil {
		s.watcher.Close()
	}
}

// Put or update item in current item
func (s *Section) Put(id string, object interface{}) error {
	return s.db.Put(object, id, s.sections...)
}

// Get single item in current item
func (s *Section) Get(id string, object interface{}) error {
	return s.db.Get(object, id, s.sections...)
}

// List subsections and items in current section
func (s *Section) List() []Record {
	return s.db.List(s.sections...)
}

// Record in database
type Record struct {
	db         *DB
	Section    []string
	Name       string
	Subsection bool
}

// RecordEvent in database
type RecordEvent struct {
	Record
	Event Event
}

// Get content of record. Returns ErrNotItem if record is subsection
func (rec *Record) Get(target interface{}) error {
	if rec.Subsection {
		return ErrNotItem
	}
	return rec.db.Get(target, rec.Name, rec.Section...)
}

// Update content of record. Returns ErrNotItem if record is subsection
func (rec *Record) Update(target interface{}) error {
	if rec.Subsection {
		return ErrNotItem
	}
	return rec.db.Put(target, rec.Name, rec.Section...)
}

// Remove content of record. Returns ErrNotItem if record is subsection
func (rec *Record) Remove() error {
	if rec.Subsection {
		return ErrNotItem
	}
	return rec.db.RemoveItem(rec.Name, rec.Section...)
}
