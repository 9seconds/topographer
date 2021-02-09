package providers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/EvilSuperstars/go-cidrman"
	"github.com/kentik/patricia"
	"github.com/kentik/patricia/uint8_tree"
)

var (
	errSoftware77DBNotFound = errors.New("ip has not been found")
)

type software77DB struct {
	v4Tree            *uint8_tree.TreeV4
	v6Tree            *uint8_tree.TreeV6
	countryCodeMap    map[uint8]string
	countryCodeMapInv map[string]uint8
}

func (s *software77DB) Lookup(addr net.IP) (string, error) {
	var (
		ok    bool
		value uint8
		err   error
	)

	v4Addr := addr.To4()

	if v4Addr != nil {
		ok, value, err = s.v4Tree.FindDeepestTag(patricia.NewIPv4AddressFromBytes(v4Addr, 32))
	} else {
		ok, value, err = s.v6Tree.FindDeepestTag(patricia.NewIPv6Address(addr.To16(), 128))
	}

	switch {
	case err != nil:
		return "", fmt.Errorf("cannot resolve ip address: %w", err)
	case !ok:
		return "", errSoftware77DBNotFound
	}

	return s.countryCodeMap[value], nil
}

func (s *software77DB) AddIPv4Range(start, end, countryCode string) error {
	countryCode = s.normalizeCountryCode(countryCode)
	if countryCode == "" {
		return nil
	}

	var startIP, endIP [4]byte

	startNum, err := strconv.Atoi(start)
	if err != nil {
		return fmt.Errorf("cannot convert a start ip: %w", err)
	}

	endNum, err := strconv.Atoi(end)
	if err != nil {
		return fmt.Errorf("cannot convert a start ip: %w", err)
	}

	binary.BigEndian.PutUint32(startIP[:], uint32(startNum))
	binary.BigEndian.PutUint32(endIP[:], uint32(endNum))

	ipnets, err := cidrman.IPRangeToIPNets(net.IP(startIP[:]), net.IP(endIP[:]))
	if err != nil {
		return fmt.Errorf("cannot build a list of ipnets: %w", err)
	}

	for _, ipnet := range ipnets {
		if err := s.add(ipnet, countryCode); err != nil {
			return err
		}
	}

	return nil
}

func (s *software77DB) AddIPv6CIDR(cidr, countryCode string) error {
	countryCode = s.normalizeCountryCode(countryCode)
	if countryCode == "" {
		return nil
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("cannot parse cidr: %w", err)
	}

	return s.add(ipnet, countryCode)
}

func (s *software77DB) add(ipnet *net.IPNet, countryCode string) error {
	intCountryCode, ok := s.countryCodeMapInv[countryCode]
	if !ok {
		intCountryCode = uint8(len(s.countryCodeMap))
		s.countryCodeMapInv[countryCode] = intCountryCode
		s.countryCodeMap[intCountryCode] = countryCode
	}

	addrLength, _ := ipnet.Mask.Size()
	addrBytes := ipnet.IP.To4()

	if addrBytes != nil {
		_, _, err := s.v4Tree.Add(patricia.NewIPv4AddressFromBytes(addrBytes, uint(addrLength)),
			intCountryCode,
			nil)

		return err
	} else {
		addrBytes = ipnet.IP.To16()

		_, _, err := s.v6Tree.Add(patricia.NewIPv6Address(addrBytes, uint(addrLength)),
			intCountryCode,
			nil)

		return err
	}
}

func (s *software77DB) normalizeCountryCode(countryCode string) string {
	countryCode = strings.ToUpper(countryCode)

    // please read comments in downloaded CSV files
	switch countryCode {
	case "ZZ", "AP", "EU":
		return ""
	case "YU":
		return "CS"
	case "FX":
		return "FR"
	case "UK":
		return "GB"
	default:
		return countryCode
	}
}

func newSoftware77DB() *software77DB {
	return &software77DB{
		v4Tree:            uint8_tree.NewTreeV4(),
		v6Tree:            uint8_tree.NewTreeV6(),
		countryCodeMap:    map[uint8]string{},
		countryCodeMapInv: map[string]uint8{},
	}
}
