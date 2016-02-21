package main

import (
	"errors"
	"strings"
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
		URL  string
		Tags map[string]struct{}
	}

	// imageDB is a tagged image database.
	// eventually a true db, for now all in-memory
	imageDB struct {
		tags   map[string]tagEntry
		images map[string]imageEntry
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

// lookupByTags returns the set of images that match all of 'include' and none
// of 'exclude'.
func (db *imageDB) lookupByTags(include, exclude []string) (urls []string, err error) {
	// if no include tags are supplied, filter the entire database
	if len(include) == 0 {
		for _, entry := range db.images {
			if entry.missingTags(exclude) {
				urls = append(urls, entry.URL)
			}
		}
		return
	}

	// Get initial set by querying a single tag. Then, of these, filter out
	// those that do not contain all of include and none of exclude.
	for url := range db.tags[include[0]].Images {
		entry := db.images[url]
		if entry.hasTags(include) && entry.missingTags(exclude) {
			urls = append(urls, entry.URL)
		}
	}
	return
}

// addImage adds an image and its tags to the database.
func (db *imageDB) addImage(url string, tags []string) error {
	if _, ok := db.images[url]; ok {
		return errImageExists
	}
	// create imageEntry without any tags, then call addTags
	db.images[url] = imageEntry{
		URL:  url,
		Tags: make(map[string]struct{}),
	}
	return db.addTags(url, tags)
}

// addTags adds a set of tags to an image.
func (db *imageDB) addTags(url string, tags []string) error {
	if _, ok := db.images[url]; !ok {
		return errImageNotExists
	}
	for _, tag := range tags {
		// create tag if it does not already exist
		if _, ok := db.tags[tag]; !ok {
			db.tags[tag] = tagEntry{
				Name:   tag,
				Images: make(map[string]struct{}),
			}
		}
		// add tag to image
		db.images[url].Tags[tag] = struct{}{}
		// add image to tag
		db.tags[tag].Images[url] = struct{}{}
	}
	return nil
}

func newImageDB() *imageDB {
	db := &imageDB{
		tags:   make(map[string]tagEntry),
		images: make(map[string]imageEntry),
	}
	for _, img := range dbData {
		db.addImage(img.Name, strings.Split(img.Tags, " "))
	}
	return db
}
