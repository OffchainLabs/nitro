package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/http/backlog/mocks"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

type HandlerSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	backlog *mocks.MockBacklog
	handler *BroadcastHandler
	res     *http.Response
}

func (s *HandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.backlog = mocks.NewMockBacklog(s.ctrl)
	s.handler = &BroadcastHandler{s.backlog}
}

func (s *HandlerSuite) TearDownTest() {
	s.ctrl.Finish()
	if s.res != nil && s.res.Body != nil {
		s.res.Body.Close()
	}
}

func (s *HandlerSuite) TestServeHTTP() {
	testcases := []struct {
		name    string
		start   arbutil.MessageIndex
		end     arbutil.MessageIndex
		indexes []arbutil.MessageIndex
		expCode int
	}{
		{
			"SingleMessage",
			10,
			10,
			[]arbutil.MessageIndex{10},
			http.StatusOK,
		},
		{
			"ManyMessages",
			10,
			12,
			[]arbutil.MessageIndex{10, 11, 12},
			http.StatusOK,
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			bm := m.CreateDummyBroadcastMessage(tc.indexes)
			s.backlog.EXPECT().Get(tc.start, tc.end).Return(bm, nil)
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?start=%d&end=%d", tc.start, tc.end), nil)
			w := httptest.NewRecorder()
			s.handler.ServeHTTP(w, r)
			res := w.Result()
			validateResponse(t, res, tc.expCode, tc.indexes)
		})
	}
}

func (s *HandlerSuite) TestServeHTTPError() {
	testcases := []struct {
		name   string
		method string
		url    string
	}{
		{
			"MethodHead",
			http.MethodHead,
			"/?start=10&end=10",
		},
		{
			"MethodPost",
			http.MethodPost,
			"/?start=10&end=10",
		},
		{
			"MethodPut",
			http.MethodPut,
			"/?start=10&end=10",
		},
		{
			"MethodPatch",
			http.MethodPatch,
			"/?start=10&end=10",
		},
		{
			"MethodDelete",
			http.MethodDelete,
			"/?start=10&end=10",
		},
		{
			"MethodConnect",
			http.MethodConnect,
			"/?start=10&end=10",
		},
		{
			"MethodOptions",
			http.MethodOptions,
			"/?start=10&end=10",
		},
		{
			"MethodTrace",
			http.MethodTrace,
			"/?start=10&end=10",
		},
		{
			"BadPath",
			http.MethodGet,
			"/wrongpath?start=10&end=10",
		},
		{
			"MissingStartParam",
			http.MethodGet,
			"/?end=10",
		},
		{
			"MissingEndParam",
			http.MethodGet,
			"/?start=10",
		},
		{
			"BadStartParam",
			http.MethodGet,
			"/?start=wrong&end=10",
		},
		{
			"BadEndParam",
			http.MethodGet,
			"/?start=10&end=wrong",
		},
		{
			"WrongParam",
			http.MethodGet,
			"/?wrong=10",
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()
			s.handler.ServeHTTP(w, r)
			res := w.Result()
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("request received status code %d but expected %d", res.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func (s *HandlerSuite) TestServeHTTPBacklogError() {
	bm := m.CreateDummyBroadcastMessage([]arbutil.MessageIndex{10, 11, 12})
	i := arbutil.MessageIndex(10)
	s.backlog.EXPECT().Get(i, i).Return(bm, errors.New("mock error"))
	r := httptest.NewRequest(http.MethodGet, "/?start=10&end=10", nil)
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, r)
	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		s.T().Errorf("request received status code %d but expected %d", res.StatusCode, http.StatusBadRequest)
	}
}

// Check that getting messages that any error from backlog results in an error

func validateResponse(t *testing.T, res *http.Response, expCode int, expIndexes []arbutil.MessageIndex) {
	if res.StatusCode != expCode {
		t.Errorf("request received status code %d but expected %d", res.StatusCode, expCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bm := &m.BroadcastMessage{}
	err = json.Unmarshal(data, bm)
	if err != nil {
		t.Fatalf("unexpected error whilst unmarshaling: %v", err)
	}

	actualIndexes := []arbutil.MessageIndex{}
	for _, msg := range bm.Messages {
		actualIndexes = append(actualIndexes, msg.SequenceNumber)
	}
	if slices.Compare(actualIndexes, expIndexes) != 0 {
		t.Errorf("request received these messages %v but expected %v", actualIndexes, expIndexes)
	}
}
