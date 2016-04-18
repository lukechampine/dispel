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
	tagEntry struct {
		Name   string
		Images map[string]struct{}
	}

	imageEntry struct {
		Hash      string
		Ext       string
		DateAdded string
		Tags      map[string]struct{}
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

func (ie imageEntry) MarshalJSON() ([]byte, error) {
	data := struct {
		Hash      string
		Ext       string
		DateAdded string
		Tags      []string
	}{ie.Hash, ie.Ext, ie.DateAdded, nil}
	for t := range ie.Tags {
		data.Tags = append(data.Tags, t)
	}
	return json.Marshal(data)
}

func (ie *imageEntry) UnmarshalJSON(b []byte) error {
	var data struct {
		Hash      string
		Ext       string
		DateAdded string
		Tags      []string
	}
	err := json.Unmarshal(b, &data)
	ie.Hash, ie.Ext, ie.DateAdded = data.Hash, data.Ext, data.DateAdded
	ie.Tags = make(map[string]struct{})
	for _, t := range data.Tags {
		ie.Tags[t] = struct{}{}
	}
	return err
}

func (te tagEntry) MarshalJSON() ([]byte, error) {
	data := struct {
		Name   string
		Images []string
	}{te.Name, nil}
	for img := range te.Images {
		data.Images = append(data.Images, img)
	}
	return json.Marshal(data)
}

func (te *tagEntry) UnmarshalJSON(b []byte) error {
	var data struct {
		Name   string
		Images []string
	}
	err := json.Unmarshal(b, &data)
	te.Name = data.Name
	te.Images = make(map[string]struct{})
	for _, img := range data.Images {
		te.Images[img] = struct{}{}
	}
	return err
}

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
				Images: make(map[string]struct{}),
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
