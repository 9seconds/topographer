package topolib_test

import (
	"testing"

	"github.com/9seconds/topographer/topolib"
	"github.com/stretchr/testify/suite"
)

type CountryCodeTestSuite struct {
	suite.Suite
}

func (suite *CountryCodeTestSuite) TestNormalizeAlpha2Code() {
	suite.Equal("RU", topolib.NormalizeAlpha2Code("ru"))
	suite.Equal("", topolib.NormalizeAlpha2Code("zz"))
	suite.Equal("", topolib.NormalizeAlpha2Code("Eu"))
	suite.Equal("", topolib.NormalizeAlpha2Code("ap"))
	suite.Equal("", topolib.NormalizeAlpha2Code("RUS"))
	suite.Equal("FR", topolib.NormalizeAlpha2Code("FX"))
	suite.Equal("FR", topolib.NormalizeAlpha2Code("FR"))
	suite.Equal("GB", topolib.NormalizeAlpha2Code("UK"))
}

func (suite *CountryCodeTestSuite) TestAlpha2ToCountryCode() {
	suite.Equal(topolib.CountryCode(0), topolib.Alpha2ToCountryCode("zz"))
	suite.Equal("RU", topolib.Alpha2ToCountryCode("ru").String())
}

func (suite *CountryCodeTestSuite) TestAlpha3ToCountryCode() {
	suite.Equal(topolib.CountryCode(0), topolib.Alpha3ToCountryCode("zz"))
	suite.Equal("RU", topolib.Alpha3ToCountryCode("rus").String())
}

func (suite *CountryCodeTestSuite) TestDetails() {
	v := topolib.Alpha2ToCountryCode("ru")

	suite.Equal("RU", v.String())
	suite.Equal("Russia", v.Details().Name.BaseLang.Common)
}

func TestCountryCode(t *testing.T) {
	suite.Run(t, &CountryCodeTestSuite{})
}
