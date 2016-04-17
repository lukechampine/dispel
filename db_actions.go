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

// Delete removes an image from the database, along with its corresponding
// file and thumbnail.
func (db *imageDB) Delete(hash string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	entry, ok := db.Images[hash]
	if !ok {
		return errImageNotExists
	}
	err := db.removeImage(entry.Hash)
	if err != nil {
		return err
	}
	// delete image + thumbnail from disk
	os.Remove(filepath.Join("static", "images", entry.Hash+entry.Ext))
	os.Remove(filepath.Join("static", "thumbnails", entry.Hash+".jpg"))

	return db.save()
}

// SetTags sets of the tags of an existing image.
func (db *imageDB) SetTags(hash string, tags []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	img, ok := db.Images[hash]
	if !ok {
		return errImageNotExists
	}
	err := db.removeImage(img.Hash)
	if err != nil {
		return err
	}
	err = db.addImage(img.Hash, img.Ext, img.DateAdded, tags)
	if err != nil {
		return err
	}
	return db.save()
}

// Upload adds an image to the database and generates a thumbnail for it. It
// returns the image's MD5 hash.
func (db *imageDB) Upload(r io.Reader, tags []string, ext string) (string, error) {
	// simultaneously copy image to disk and calculate md5 hash
	tmpFile, err := ioutil.TempFile(os.TempDir(), "dispel")
	if err != nil {
		return "", err
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
		return "", err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	// create thumbnail
	thumbFile, err := os.Create(filepath.Join("static", "thumbnails", hash+".jpg"))
	if err != nil {
		return "", err
	}
	defer thumbFile.Close()
	thumb := resize.Thumbnail(150, 150, img, resize.MitchellNetravali)
	err = jpeg.Encode(thumbFile, thumb, nil)
	if err != nil {
		os.Remove(filepath.Join("static", "thumbnails", hash+".jpg"))
		return "", err
	}

	// move image file to static dir
	err = os.Rename(tmpFile.Name(), filepath.Join("static", "images", hash+ext))
	if err != nil {
		return "", err
	}

	// add image to database
	db.mu.Lock()
	defer db.mu.Unlock()
	date := time.Now().Format("Mon Jan 02 15:04:05 EST 2006")
	err = db.addImage(hash, ext, date, tags)
	if err != nil && err != errImageExists {
		os.Remove(filepath.Join("static", "images", hash+ext))
		os.Remove(filepath.Join("static", "thumbnails", hash+".jpg"))
		return "", err
	}

	err = db.save()
	if err != nil {
		return "", err
	}
	return hash, nil
}
