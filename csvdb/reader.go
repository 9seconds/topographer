package csvdb

import (
	"encoding/csv"
	"io"

	"github.com/juju/errors"
)

type recordMaker func([]string) (*Record, error)

type CSVReader struct {
	reader     *csv.Reader
	makeRecord recordMaker
}

func (cr *CSVReader) Read() (*Record, error) {
	data, err := cr.reader.Read()
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read new record")
	}

	record, err := cr.makeRecord(data)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot build new record")
	}

	return record, nil
}

func NewCSVReader(filefp io.Reader, makeRecord recordMaker) *CSVReader {
	reader := csv.NewReader(filefp)
	reader.ReuseRecord = true

	return &CSVReader{reader, makeRecord}
}
