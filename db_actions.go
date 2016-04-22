package main

import (
	"crypto/md5"
	"encoding/hex"
	"image"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	// register these image formats
	_ "image/gif"
	"image/jpeg" // need full import, since we write jpeg thumbnails
	_ "image/png"

	"github.com/nfnt/resize"
)

func currentTime() string { return time.Now().Format("Mon Jan 02 15:04:05 EST 2006") }

// QueueDelete adds an image to the delete queue.
func (db *imageDB) QueueDelete(hash string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	entry, ok := db.Images[hash]
	if !ok {
		return errImageNotExists
	}
	db.Queue = append(db.Queue, queueItem{
		Action:     actionDelete,
		imageEntry: entry,
	})
	return db.save()
}

// QueueSetTags adds an image to the tags queue.
func (db *imageDB) QueueSetTags(hash string, tags []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	entry, ok := db.Images[hash]
	if !ok {
		return errImageNotExists
	}
	entry.Tags = make(stringSet)
	for _, t := range tags {
		entry.Tags[t] = struct{}{}
	}
	db.Queue = append(db.Queue, queueItem{
		Action:     actionSetTags,
		imageEntry: entry,
	})
	return db.save()
}

// QueueUpload adds an image to the upload queue and generates a thumbnail for
// it. It returns the image's MD5 hash.
func (db *imageDB) QueueUpload(r io.Reader, tags []string, ext string) error {
	// simultaneously copy image to disk and calculate md5 hash
	tmpFile, err := ioutil.TempFile(os.TempDir(), "dispel")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	hasher := md5.New()
	img, _, err := image.Decode(
		io.TeeReader(
			r, // decode file data
			io.MultiWriter(
				tmpFile, // also write to disk
				hasher,  // also write to hasher
			),
		),
	)
	if err != nil {
		return err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	db.mu.RLock()
	curEntry, exists := db.Images[hash]
	db.mu.RUnlock()
	if exists {
		// if image was already uploaded, convert to setTags action instead,
		// adding any unseen tags.
		added, _ := curEntry.Tags.diff(db.expandAliases(toStringSet(tags)))
		newTags := append(fromStringSet(curEntry.Tags), added...)
		return db.QueueSetTags(hash, newTags)
	}

	// create thumbnail
	thumbFile, err := os.Create(filepath.Join("queue", hash+"_thumb.jpg"))
	if err != nil {
		return err
	}
	defer thumbFile.Close()
	thumb := resize.Thumbnail(150, 150, img, resize.MitchellNetravali)
	err = jpeg.Encode(thumbFile, thumb, nil)
	if err != nil {
		os.Remove(filepath.Join("queue", hash+"_thumb.jpg"))
		return err
	}

	// move image file to queue dir
	err = os.Rename(tmpFile.Name(), filepath.Join("queue", hash+ext))
	if err != nil {
		return err
	}

	// add image to queue
	db.mu.Lock()
	defer db.mu.Unlock()
	db.Queue = append(db.Queue, queueItem{
		Action: actionUpload,
		imageEntry: imageEntry{
			Hash:      hash,
			Ext:       ext,
			DateAdded: currentTime(),
			Tags:      toStringSet(tags),
		},
	})
	return db.save()
}
