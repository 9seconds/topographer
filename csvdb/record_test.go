package csvdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordOK(t *testing.T) {
	record, err := NewRecord("RU", "Nizhniy Novgorod", "93.94.95.1", "93.94.95.255")
	assert.Nil(t, err)
	assert.Equal(t, record.Country, "ru")
	assert.Equal(t, record.City, "Nizhniy Novgorod")
	assert.Equal(t, record.StartIP, "93.94.95.1")
	assert.Equal(t, record.FinishIP, "93.94.95.255")

	subnets, err := record.GetSubnets()
	assert.Nil(t, err)
	assert.Len(t, subnets, 8)
}

func TestRecordUnknownCountry(t *testing.T) {
	for _, country := range []string{"zz", "zZ", "Zz", "ZZ"} {
		_, err := NewRecord(country, "Nizhniy Novgorod", "93.94.95.1", "93.94.95.255")
		assert.NotNil(t, err)
	}
}

func TestRecordIncorrectIP(t *testing.T) {
	_, err := NewRecord("ru", "Moscow", "x", "93.94.95.255")
	assert.NotNil(t, err)

	_, err = NewRecord("ru", "Moscow", "93.94.95.255", "t")
	assert.NotNil(t, err)
}
