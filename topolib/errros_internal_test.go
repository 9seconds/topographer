package topolib

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HTTPErrorTestSuite struct {
	suite.Suite

	e *httpError
}

func (suite *HTTPErrorTestSuite) SetupTest() {
	suite.e = &httpError{}
}

func (suite *HTTPErrorTestSuite) TestNil() {
	var err *httpError

	suite.Equal("", err.Message())
	suite.Equal("", err.Err())
	suite.Equal(http.StatusInternalServerError, err.StatusCode())
	suite.Nil(err.Unwrap())
	suite.Nil(errors.Unwrap(err))
	suite.Equal("", err.Error())

	data, e := json.Marshal(err)

	suite.NoError(e)
	suite.JSONEq("null", string(data))
}

func (suite *HTTPErrorTestSuite) TestMessage() {
	suite.Equal("", suite.e.Message())

	suite.e.message = "hello"

	suite.Equal("hello", suite.e.Message())
}

func (suite *HTTPErrorTestSuite) TestErr() {
	suite.Equal("", suite.e.Err())

	suite.e.err = io.EOF

	suite.Equal("EOF", suite.e.Err())
}

func (suite *HTTPErrorTestSuite) TestStatusCode() {
	suite.Equal(http.StatusInternalServerError, suite.e.StatusCode())

	suite.e.statusCode = http.StatusOK

	suite.Equal(http.StatusOK, suite.e.StatusCode())
}

func (suite *HTTPErrorTestSuite) TestUnwrap() {
	suite.Nil(suite.e.Unwrap())
	suite.Nil(errors.Unwrap(suite.e))

	suite.e.err = io.EOF

	suite.Equal(io.EOF, suite.e.Unwrap())
	suite.Equal(io.EOF, errors.Unwrap(suite.e))
	suite.True(errors.Is(suite.e, io.EOF))
}

func (suite *HTTPErrorTestSuite) TestError() {
	suite.EqualError(suite.e, "")

	suite.e.message = "message"

	suite.EqualError(suite.e, "message")

	suite.e.message = ""
	suite.e.err = io.EOF

	suite.EqualError(suite.e, "EOF")

	suite.e.message = "msg"

	suite.Contains(suite.e.Error(), "msg")
	suite.Contains(suite.e.Error(), "EOF")
}

func (suite *HTTPErrorTestSuite) TestJSON() {
	data, err := json.Marshal(suite.e)

	suite.NoError(err)
	suite.JSONEq(`{"error": {"message": "", "context": ""}}`, string(data))

	suite.e.message = "Msg"
	data, err = json.Marshal(suite.e)

	suite.NoError(err)
	suite.JSONEq(`{"error": {"message": "Msg", "context": ""}}`, string(data))

	suite.e.err = io.EOF
	data, err = json.Marshal(suite.e)

	suite.NoError(err)
	suite.JSONEq(`{"error": {"message": "Msg", "context": "EOF"}}`, string(data))

	suite.e.message = ""
	data, err = json.Marshal(suite.e)

	suite.NoError(err)
	suite.JSONEq(`{"error": {"message": "", "context": "EOF"}}`, string(data))
}

func TestHTTPError(t *testing.T) {
	suite.Run(t, &HTTPErrorTestSuite{})
}
