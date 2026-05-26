package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDictionary(t *testing.T) {
	dict := NewDictionary()
	assert.NotNil(t, dict.LibMap)
	assert.Equal(t, 0, len(dict.LibMap))
}

func TestDictionary_PutProperty(t *testing.T) {
	dict := NewDictionary()
	dict.PutProperty("key1", "value1")
	assert.Equal(t, []string{"value1"}, dict.LibMap["key1"])
}

func TestDictionary_PutProperty_MultipleValues(t *testing.T) {
	dict := NewDictionary()
	dict.PutProperty("key1", "value1")
	dict.PutProperty("key1", "value2")
	assert.Equal(t, []string{"value1", "value2"}, dict.LibMap["key1"])
}

func TestDictionary_GetProperty(t *testing.T) {
	dict := NewDictionary()
	dict.PutProperty("key1", "value1")
	result := dict.GetProperty("key1")
	assert.Equal(t, []string{"value1"}, result)
}

func TestDictionary_GetProperty_NotExist(t *testing.T) {
	dict := NewDictionary()
	result := dict.GetProperty("nonexistent")
	assert.Nil(t, result)
}
