package providers

import (
	"encoding/binary"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/csvdb"
	"github.com/juju/errors"
)

const (
	software77URL = "http://software77.net/geo-ip/?DL=1"

	software77IdxStartIP  = 0
	software77IdxFinishIP = 1
	software77IdxCountry  = 4
)

// Software77 presents a structure for software77 provider.
type Software77 struct {
	CSVDBProvider
}

// Update updates database.
func (s77 *Software77) Update() (bool, error) {
	rawFile, err := s77.downloadURL(software77URL)
	if err != nil {
		return false, errors.Annotate(err, "Cannot download software77")
	}
	defer func() {
		rawFile.Close()           // nolint
		os.Remove(rawFile.Name()) // nolint
	}()

	return s77.saveFile(rawFile)
}

// NewSoftware77 creates new instance of Software77 provider.
func NewSoftware77(conf *config.Config) *Software77 {
	return &Software77{
		CSVDBProvider: CSVDBProvider{
			Provider: Provider{
				directory:       conf.Directory,
				dbname:          "software77",
				downloadTimeout: 3 * time.Minute,
				precision:       conf.Precision,
				updateLock:      &sync.RWMutex{},
			},
			makeRecord: func(data []string) (*csvdb.Record, error) {
				startIP, err := int2IP(data[software77IdxStartIP])
				if err != nil {
					return nil, err
				}
				finishIP, err := int2IP(data[software77IdxFinishIP])
				if err != nil {
					return nil, err
				}

				return csvdb.NewRecord(
					data[software77IdxCountry],
					"", // software77 has no city, only country
					startIP,
					finishIP)
			},
		},
	}
}

func int2IP(strIP string) (string, error) {
	ipAsNumber, err := strconv.ParseUint(strIP, 10, 32)
	if err != nil {
		return "", errors.Annotatef(err, "Cannot convert %s to number", strIP)
	}

	ipData := make(net.IP, 4)
	binary.BigEndian.PutUint32(ipData, uint32(ipAsNumber))

	return ipData.String(), nil
}
