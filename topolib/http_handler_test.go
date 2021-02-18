package topolib_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/9seconds/topographer/topolib"
	"github.com/qri-io/jsonschema"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var (
	jsonSchemaGETResolve = func() *jsonschema.Schema {
		data := `{
          "type": "object",
          "required": [
            "result"
          ],
          "additionalProperties": false,
          "properties": {
            "result": {
              "type": "object",
              "required": [
                "ip",
                "country",
                "city",
                "details"
              ],
              "additionalProperties": false,
              "properties": {
                "ip": {
                  "anyOf": [
                    {
                      "type": "string",
                      "format": "ipv4",
                      "minLength": 7,
                      "maxLength": 15
                    },
                    {
                      "type": "string",
                      "format": "ipv6",
                      "minLength": 2,
                      "maxLength": 39
                    }
                  ]
                },
                "country": {
                  "type": "object",
                  "additionalProperties": false,
                  "required": [
                    "alpha2_code",
                    "alpha3_code",
                    "common_name",
                    "official_name"
                  ],
                  "properties": {
                    "alpha2_code": {
                      "anyOf": [
                        {
                          "type": "string",
                          "maxLength": 0
                        },
                        {
                          "type": "string",
                          "minLength": 2,
                          "maxLength": 2
                        }
                      ]
                    },
                    "alpha3_code": {
                      "anyOf": [
                        {
                          "type": "string",
                          "maxLength": 0
                        },
                        {
                          "type": "string",
                          "minLength": 3,
                          "maxLength": 3
                        }
                      ]
                    },
                    "common_name": {
                      "type": "string"
                    },
                    "official_name": {
                      "type": "string"
                    }
                  }
                },
                "city": {
                  "type": "string"
                },
                "details": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "required": [
                      "provider_name",
                      "country_code",
                      "city"
                    ],
                    "additionalProperties": false,
                    "properties": {
                      "provider_name": {
                        "type": "string",
                        "minLength": 1
                      },
                      "country_code": {
                        "anyOf": [
                          {
                            "type": "string",
                            "maxLength": 0
                          },
                          {
                            "type": "string",
                            "minLength": 2,
                            "maxLength": 2
                          }
                        ]
                      },
                      "city": {
                        "type": "string"
                      }
                    }
                  }
                }
              }
            }
          }
        }`

		rv := &jsonschema.Schema{}
		if err := json.Unmarshal([]byte(data), rv); err != nil {
			panic(err)
		}

		return rv
	}()

	jsonSchemaGETStats = func() *jsonschema.Schema {
		data := `{
          "type": "object",
          "required": [
            "results"
          ],
          "additionalProperties": false,
          "properties": {
            "results": {
              "type": "array",
              "items": {
                "type": "object",
                "required": [
                  "name",
                  "last_updated",
                  "last_used",
                  "success_count",
                  "failure_count"
                ],
                "additionalProperties": false,
                "properties": {
                  "name": {
                    "type": "string",
                    "minLength": 1
                  },
                  "last_used": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "last_updated": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "success_count": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "failure_count": {
                    "type": "integer",
                    "minimum": 0
                  }
                }
              }
            }
          }
        }`

		rv := &jsonschema.Schema{}
		if err := json.Unmarshal([]byte(data), rv); err != nil {
			panic(err)
		}

		return rv
	}()

	jsonSchemaPOST = func() *jsonschema.Schema {
		data := `{
          "type": "object",
          "required": [
            "results"
          ],
          "additionalProperties": false,
          "properties": {
            "results": {
              "type": "array",
              "items": {
                "type": "object",
                "required": [
                  "ip",
                  "country",
                  "city",
                  "details"
                ],
                "additionalProperties": false,
                "properties": {
                  "ip": {
                    "anyOf": [
                      {
                        "type": "string",
                        "format": "ipv4",
                        "minLength": 7,
                        "maxLength": 15
                      },
                      {
                        "type": "string",
                        "format": "ipv6",
                        "minLength": 2,
                        "maxLength": 39
                      }
                    ]
                  },
                  "country": {
                    "type": "object",
                    "additionalProperties": false,
                    "required": [
                      "alpha2_code",
                      "alpha3_code",
                      "common_name",
                      "official_name"
                    ],
                    "properties": {
                      "alpha2_code": {
                        "anyOf": [
                          {
                            "type": "string",
                            "maxLength": 0
                          },
                          {
                            "type": "string",
                            "minLength": 2,
                            "maxLength": 2
                          }
                        ]
                      },
                      "alpha3_code": {
                        "anyOf": [
                          {
                            "type": "string",
                            "maxLength": 0
                          },
                          {
                            "type": "string",
                            "minLength": 3,
                            "maxLength": 3
                          }
                        ]
                      },
                      "common_name": {
                        "type": "string"
                      },
                      "official_name": {
                        "type": "string"
                      }
                    }
                  },
                  "city": {
                    "type": "string"
                  },
                  "details": {
                    "type": "array",
                    "items": {
                      "type": "object",
                      "required": [
                        "provider_name",
                        "country_code",
                        "city"
                      ],
                      "additionalProperties": false,
                      "properties": {
                        "provider_name": {
                          "type": "string",
                          "minLength": 1
                        },
                        "country_code": {
                          "anyOf": [
                            {
                              "type": "string",
                              "maxLength": 0
                            },
                            {
                              "type": "string",
                              "minLength": 2,
                              "maxLength": 2
                            }
                          ]
                        },
                        "city": {
                          "type": "string"
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }`

		rv := &jsonschema.Schema{}
		if err := json.Unmarshal([]byte(data), rv); err != nil {
			panic(err)
		}

		return rv
	}()
)

type HTTPHanderTestSuite struct {
	suite.Suite

	h            http.Handler
	providerMock *ProviderMock
	loggerMock   *LoggerMock
	resp         *httptest.ResponseRecorder
}

func (suite *HTTPHanderTestSuite) SetupTest() {
	suite.providerMock = &ProviderMock{}
	suite.loggerMock = &LoggerMock{}

	suite.providerMock.On("Name").Return("providerMock").Maybe()
	suite.loggerMock.On("UpdateInfo", mock.Anything, mock.Anything).Maybe()
	suite.loggerMock.On("LookupError", mock.Anything, mock.Anything, mock.Anything).Maybe()
	suite.loggerMock.On("UpdateError", mock.Anything, mock.Anything).Maybe()

	providers := []topolib.Provider{suite.providerMock}

	topo, err := topolib.NewTopographer(providers, suite.loggerMock, 10)
	if err != nil {
		panic(err)
	}

	suite.h = topo
	suite.resp = httptest.NewRecorder()
}

func (suite *HTTPHanderTestSuite) TearDownTest() {
	suite.providerMock.AssertExpectations(suite.T())
	suite.loggerMock.AssertExpectations(suite.T())
}

func (suite *HTTPHanderTestSuite) TestIncorrectMethod() {
	suite.h.ServeHTTP(suite.resp, httptest.NewRequest("PATCH", "/", nil))

	suite.Equal(http.StatusMethodNotAllowed, suite.resp.Code)
}

func (suite *HTTPHanderTestSuite) TestGetOk() {
	result := topolib.ProviderLookupResult{
		CountryCode: topolib.Alpha2ToCountryCode("RU"),
		City:        "Nizhniy Novgorod",
	}
	ip := net.ParseIP("192.168.1.1").To16()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:5678"

	suite.providerMock.On("Lookup", mock.Anything, ip).Return(result, nil).Once()

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusOK, suite.resp.Code)

	errs, err := jsonSchemaGETResolve.ValidateBytes(context.Background(),
		suite.resp.Body.Bytes())

	suite.NoError(err)
	suite.Empty(errs)
	suite.Contains(suite.resp.Body.String(), "192.168.1.1")
	suite.Contains(suite.resp.Body.String(), "RU")
	suite.Contains(suite.resp.Body.String(), "RUS")
	suite.Contains(suite.resp.Body.String(), "Nizhniy Novgorod")
}

func (suite *HTTPHanderTestSuite) TestGetUnkownPath() {
	req := httptest.NewRequest("GET", "/lalala", nil)
	req.RemoteAddr = "192.168.1.1:5678"

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusNotFound, suite.resp.Code)
}

func (suite *HTTPHanderTestSuite) TestGetStats() {
	req := httptest.NewRequest("GET", "/stats/", nil)
	req.RemoteAddr = "192.168.1.1:5678"

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusOK, suite.resp.Code)

	errs, err := jsonSchemaGETStats.ValidateBytes(context.Background(),
		suite.resp.Body.Bytes())

	suite.NoError(err)
	suite.Empty(errs)
}

func (suite *HTTPHanderTestSuite) TestPostUnknownPath() {
	req := httptest.NewRequest("POST", "/lalala", nil)
	req.RemoteAddr = "192.168.1.1:5678"

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusNotFound, suite.resp.Code)
}

func (suite *HTTPHanderTestSuite) TestPostUnsupportedMediaType() {
	req := httptest.NewRequest("POST", "/", strings.NewReader("{}"))

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusUnsupportedMediaType, suite.resp.Code)
}

func (suite *HTTPHanderTestSuite) TestPostBadRequest() {
	req := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	req.Header.Add("Content-Type", "application/json")

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusBadRequest, suite.resp.Code)
}

func (suite *HTTPHanderTestSuite) TestPostOk() {
	req := httptest.NewRequest("POST",
		"/",
		strings.NewReader(`{"ips": ["192.168.1.1"]}`))
	result := topolib.ProviderLookupResult{
		CountryCode: topolib.Alpha2ToCountryCode("RU"),
		City:        "Nizhniy Novgorod",
	}
	ip := net.ParseIP("192.168.1.1").To16()

	req.Header.Add("Content-Type", "application/json")

	suite.providerMock.On("Lookup", mock.Anything, ip).Return(result, nil).Once()

	suite.h.ServeHTTP(suite.resp, req)

	suite.Equal(http.StatusOK, suite.resp.Code)

	errs, err := jsonSchemaPOST.ValidateBytes(context.Background(), suite.resp.Body.Bytes())

	suite.NoError(err)
	suite.Empty(errs)
	suite.Contains(suite.resp.Body.String(), "192.168.1.1")
	suite.Contains(suite.resp.Body.String(), "RU")
	suite.Contains(suite.resp.Body.String(), "RUS")
	suite.Contains(suite.resp.Body.String(), "Nizhniy Novgorod")
}

func TestHTTPHandler(t *testing.T) {
	suite.Run(t, &HTTPHanderTestSuite{})
}
