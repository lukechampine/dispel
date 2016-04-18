package main

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

const (
	actionUpload  = "upload"
	actionSetTags = "set tags"
	actionDelete  = "delete"
)

var (
	errImageExists    = errors.New("image already exists")
	errImageNotExists = errors.New("image does not exist")
)

type (
	stringSet map[string]struct{}

	tagEntry struct {
		Name   string
		Images stringSet
	}

	imageEntry struct {
		Hash      string
		Ext       string
		DateAdded string
		Tags      stringSet
	}

	// a queueItem is a user action awaiting review
	queueItem struct {
		Action string
		imageEntry
	}

	// imageDB is a tagged image database.
	// eventually a true db, for now all in-memory
	imageDB struct {
		Tags    map[string]tagEntry
		Images  map[string]imageEntry
		Aliases map[string]string

		Queue []queueItem

		mu sync.RWMutex
	}
)

func toStringSet(strs []string) stringSet {
	ss := make(stringSet)
	for _, str := range strs {
		ss[str] = struct{}{}
	}
	return ss
}

func fromStringSet(ss stringSet) []string {
	var strs []string
	for str := range ss {
		strs = append(strs, str)
	}
	return strs
}

func (ss stringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(fromStringSet(ss))
}

func (ss *stringSet) UnmarshalJSON(b []byte) error {
	var strs []string
	err := json.Unmarshal(b, &strs)
	*ss = toStringSet(strs)
	return err
}

// checkTags returns true if the existence of each tag in the imageEntry
// accords with check.
func (ie imageEntry) checkTags(tags []string, check bool) bool {
	for _, tag := range tags {
		if _, ok := ie.Tags[tag]; ok != check {
			return false
		}
	}
	return true
}

// hasTags returns true if the imageEntry contains all of the specified tags.
func (ie imageEntry) hasTags(tags []string) bool { return ie.checkTags(tags, true) }

// missingTags returns true if the imageEntry contains none of the specified tags.
func (ie imageEntry) missingTags(tags []string) bool { return ie.checkTags(tags, false) }

// lookupByTags returns the set of images that match all of 'include' and none
// of 'exclude'.
func (db *imageDB) lookupByTags(include, exclude []string) (imgs []imageEntry, err error) {
	// expand tag aliases
	for i, tag := range include {
		if alias, ok := db.Aliases[tag]; ok {
			include[i] = alias
		}
	}
	for i, tag := range exclude {
		if alias, ok := db.Aliases[tag]; ok {
			exclude[i] = alias
		}
	}

	// if no include tags are supplied, filter the entire database
	if len(include) == 0 {
		for _, entry := range db.Images {
			if entry.missingTags(exclude) {
				imgs = append(imgs, entry)
			}
		}
		return
	}

	// Get initial set by querying a single tag. Then, of these, filter out
	// those that do not contain all of include and none of exclude.
	for url := range db.Tags[include[0]].Images {
		entry := db.Images[url]
		if entry.hasTags(include) && entry.missingTags(exclude) {
			imgs = append(imgs, entry)
		}
	}
	return
}

// addImage adds an image and its tags to the database.
func (db *imageDB) addImage(entry imageEntry) error {
	if _, ok := db.Images[entry.Hash]; ok {
		return errImageExists
	}
	db.Images[entry.Hash] = entry
	for tag := range entry.Tags {
		// expand alias, if there is one
		if alias, ok := db.Aliases[tag]; ok {
			tag = alias
		}

		// create tag if it does not already exist
		if _, ok := db.Tags[tag]; !ok {
			db.Tags[tag] = tagEntry{
				Name:   tag,
				Images: make(stringSet),
			}
		}
		// add image to tag
		db.Tags[tag].Images[entry.Hash] = struct{}{}
	}
	return nil
}

// removeImage deletes an image from the database. If a tag only applied to
// that image, the tag is also deleted.
func (db *imageDB) removeImage(hash string) error {
	img, ok := db.Images[hash]
	if !ok {
		return errImageNotExists
	}
	// delete tags
	for t := range img.Tags {
		tag, ok := db.Tags[t]
		if !ok {
			continue
		}
		delete(tag.Images, hash)
		if len(tag.Images) == 0 {
			delete(db.Tags, t)
		}
	}
	// delete image entry
	delete(db.Images, hash)
	return nil
}

func (db *imageDB) save() error {
	f, err := os.Create("imagedb.json")
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.MarshalIndent(db, "", "\t")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

func newImageDB(dbpath string) (*imageDB, error) {
	f, err := os.OpenFile(dbpath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	db := &imageDB{
		Tags:    make(map[string]tagEntry),
		Images:  make(map[string]imageEntry),
		Aliases: make(map[string]string),
	}
	err = json.NewDecoder(f).Decode(&db)
	return db, err
}
