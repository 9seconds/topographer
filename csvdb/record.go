package csvdb

import (
	"net"
	"strings"

	cidrman "github.com/EvilSuperstars/go-cidrman"

	"github.com/juju/errors"
)

// Record presents an extracted data from CSV record.
type Record struct {
	Country  string
	City     string
	StartIP  string
	FinishIP string
}

// GetSubnets returns non-overlapping subnets of the given Record.
func (r *Record) GetSubnets() (subnets []string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			switch x := rec.(type) {
			case string:
				err = errors.Annotate(errors.New(x), "Incorrect subnets")
			case error:
				err = errors.Annotate(x, "Incorrect subnets")
			}
		}
	}()

	subnets, err = cidrman.IPRangeToCIDRs(r.StartIP, r.FinishIP)
	return
}

// NewRecord creates new CSV record.
func NewRecord(country, city, startIP, finishIP string) (*Record, error) {
	country = strings.ToLower(country)
	if country == "zz" {
		return nil, errors.New("Country is unknown")
	}

	if !ipOk(startIP) {
		return nil, errors.New("Start IP is not correct")
	}
	if !ipOk(finishIP) {
		return nil, errors.New("Finish IP is not correct")
	}

	return &Record{country, city, startIP, finishIP}, nil
}

func ipOk(ip string) bool {
	parsed := net.ParseIP(ip)

	return parsed != nil && parsed.To4() != nil
}
