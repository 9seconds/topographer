package csvdb

import (
	"encoding/csv"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/juju/errors"
)

// RecordMaker is a type which converts parsed CSV record to the Record instance.
type RecordMaker func([]string) (*Record, error)

// CSVReader is a wrapper over csv.Reader to convert each row into Record instance.
type CSVReader struct {
	reader     *csv.Reader
	makeRecord RecordMaker
}

func (cr *CSVReader) Read() (*Record, error) {
	data, err := cr.next()
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, errors.Annotate(err, "Cannot read new record")
	}

	record, err := cr.makeRecord(data)
	if err != nil {
		log.WithFields(log.Fields{
			"data": data,
			"err":  err,
		}).Debug("Cannot parse record")
		record = nil
	}

	return record, nil
}

func (cr *CSVReader) next() (data []string, err error) {
	for err == nil && len(data) == 0 {
		data, err = cr.reader.Read()
	}

	return
}

// NewCSVReader converts given io.Reader instance into CSVReader.
func NewCSVReader(filefp io.Reader, makeRecord RecordMaker) *CSVReader {
	reader := csv.NewReader(filefp)
	reader.ReuseRecord = true
	reader.Comment = '#'

	return &CSVReader{reader, makeRecord}
}
