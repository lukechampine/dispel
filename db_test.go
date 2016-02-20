package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddImage(t *testing.T) {
	assert, require := assert.New(t), require.New(t)
	db := newImageDB()

	err := db.addImage("foo", []string{"bar", "baz"})
	require.Nil(err)

	err = db.addImage("foo", []string{"bar", "baz"})
	assert.Equal(err, errImageExists)

	urls, err := db.lookupByTags([]string{"bar"}, nil)
	assert.Nil(err)
	assert.Contains(urls, "foo")

	urls, err = db.lookupByTags([]string{"baz"}, nil)
	assert.Nil(err)
	assert.Contains(urls, "foo")
}

func TestAddTags(t *testing.T) {
	assert, require := assert.New(t), require.New(t)
	db := newImageDB()

	err := db.addTags("foo", []string{"bar", "baz"})
	assert.Equal(err, errImageNotExists)

	err = db.addImage("foo", nil)
	require.Nil(err)

	err = db.addTags("foo", []string{"bar", "baz"})
	assert.Nil(err)

	urls, err := db.lookupByTags([]string{"bar"}, nil)
	assert.Nil(err)
	assert.Contains(urls, "foo")

	urls, err = db.lookupByTags([]string{"baz"}, nil)
	assert.Nil(err)
	assert.Contains(urls, "foo")
}

func TestLookupByTags(t *testing.T) {
	assert, require := assert.New(t), require.New(t)
	db := newImageDB()

	urls, err := db.lookupByTags([]string{"bar"}, nil)
	assert.Nil(err)
	assert.Empty(urls)

	err = db.addImage("foo", []string{"bar", "baz"})
	require.Nil(err)

	urls, err = db.lookupByTags([]string{"bar"}, nil)
	assert.Nil(err)
	assert.Contains(urls, "foo")

	urls, err = db.lookupByTags([]string{"baz"}, nil)
	assert.Nil(err)
	assert.Contains(urls, "foo")
}

func TestParseTags(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		tagQuery         string
		include, exclude []string
	}{
		{"", nil, nil},
		{"-", nil, nil},
		{"-+--++---+++", nil, nil},
		{"foo", []string{"foo"}, nil},
		{"+foo", []string{"foo"}, nil},
		{"foo+-", []string{"foo"}, nil},
		{"foo+bar", []string{"foo", "bar"}, nil},
		{"foo+-bar", []string{"foo"}, []string{"bar"}},
		{"-bar", nil, []string{"bar"}},
	}
	for _, test := range tests {
		inc, ex := parseTags(test.tagQuery)
		assert.Equal(inc, test.include)
		assert.Equal(ex, test.exclude)
	}
}
