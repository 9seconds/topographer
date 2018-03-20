package csvdb

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReaderOK(t *testing.T) {
	content := bytes.NewBufferString(`#lalala
# comment
127.0.0.1,127.0.0.2,ru
# comment
`)
	reader := NewCSVReader(content, func(data []string) (*Record, error) {
		assert.Len(t, data, 3)

		return NewRecord(data[2], "", data[0], data[1])
	})
	item, err := reader.Read()

	assert.Nil(t, err)
	assert.Equal(t, item.Country, "ru")
	assert.Equal(t, item.City, "")
	assert.Equal(t, item.StartIP, "127.0.0.1")
	assert.Equal(t, item.FinishIP, "127.0.0.2")

	_, err = reader.Read()
	assert.Equal(t, err, io.EOF)
}

func TestReaderCannotParse(t *testing.T) {
	content := bytes.NewBufferString(`#lalala
# comment
127.0.0.1,127.0.0.2,ru
# comment
`)
	reader := NewCSVReader(content, func(data []string) (*Record, error) {
		assert.Len(t, data, 3)

		return NewRecord(data[2], "", "x", "y")
	})
	item, err := reader.Read()

	assert.Nil(t, err)
	assert.Nil(t, item)
}

func TestReaderIncorrectCSV(t *testing.T) {
	content := bytes.NewBufferString(`#lalala
# comment
"
# comment
`)
	reader := NewCSVReader(content, func(data []string) (*Record, error) {
		assert.Len(t, data, 3)

		return NewRecord(data[2], "", "x", "y")
	})
	_, err := reader.Read()

	assert.NotNil(t, err)
}
