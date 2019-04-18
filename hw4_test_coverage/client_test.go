package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Row struct {
	ID            int    `xml:"id"`
	GUID          string `xml:"guid"`
	IsActive      bool   `xml:"isActive"`
	Balance       string `xml:"balance"`
	Picture       string `xml:"picture"`
	Age           int    `xml:"age"`
	EyeColor      string `xml:"eyeColor"`
	FirstName     string `xml:"first_name"`
	LastName      string `xml:"last_name"`
	Gender        string `xml:"gender"`
	Company       string `xml:"company"`
	Email         string `xml:"email"`
	Phone         string `xml:"phone"`
	About         string `xml:"about"`
	Registered    string `xml:"registered"`
	FavoriteFruit string `xml:"favoriteFruit"`
}

type Rows struct {
	Version string `xml:"version,attr"`
	List    []Row  `xml:"row"`
}

// create []User from dataset.xml
func usersFromXML(fileName string) ([]User, error) {
	// open xml file
	xmlFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	// read xml into Rows structure
	rows := new(Rows)
	byteValue, _ := ioutil.ReadAll(xmlFile)
	err = xml.Unmarshal(byteValue, &rows)
	if err != nil {
		return nil, err
	}

	// create and fullfil []User
	users := make([]User, 0)
	for _, row := range rows.List {
		user := User{
			Id:     row.ID,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		}
		users = append(users, user)
	}
	return users, nil
}

// Handles request from SearchClient
func handlerTestServer(w http.ResponseWriter, r *http.Request) {

	// get SearchRequest from r *http.Request
	limit, err := strconv.Atoi(r.URL.Query()["limit"][0])
	offset, err := strconv.Atoi(r.URL.Query()["offset"][0])
	query := r.URL.Query()["query"][0]
	orderFieldr := r.URL.Query()["order_field"][0]
	orderBy, err := strconv.Atoi(r.URL.Query()["order_by"][0])

	sr := SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderFieldr,
		OrderBy:    orderBy,
	}

	// check SearchRequest field
	if sr.OrderField == "" {
		sr.OrderField = "Name"
	}

	// get users from a datasource
	users, err := usersFromXML("dataset.xml")
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	// get Users regarding to sr(SearchRequest)
	outputUsers := make([]User, 0)
	for _, user := range users {
		if strings.Contains(user.Name, sr.Query) || strings.Contains(user.About, sr.Query) {
			outputUsers = append(outputUsers, user)
		}
	}

	// outputUsers -> json
	resultJSON, err := json.Marshal(outputUsers)

	// put outputUsers into a response body
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resultJSON)

}

type TestInput struct {
	sc *SearchClient
	sr *SearchRequest
}
type TestResult struct {
	sr  *SearchResponse
	err error
}
type TestCase struct {
	Input  *TestInput
	Result *TestResult
}

func TestSearchClientUserSearch(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Input: &TestInput{
				sc: &SearchClient{
					AccessToken: "Authorized",
				},
				sr: &SearchRequest{
					Limit:      26,
					Offset:     1,
					Query:      "Boyd Wolf",
					OrderField: "",
					OrderBy:    0,
				},
			},
			Result: &TestResult{
				sr: &SearchResponse{
					Users: []User{
						User{
							Id:     0,
							Name:   "Boyd Wolf",
							Age:    22,
							Gender: "male",
							About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
						},
					},
					NextPage: false,
				},
				err: nil,
			},
		},
		TestCase{
			Input: &TestInput{
				sc: &SearchClient{
					AccessToken: "Authorized",
				},
				sr: &SearchRequest{
					Limit:      1,
					Offset:     0,
					Query:      "Nulla",
					OrderField: "",
					OrderBy:    0,
				},
			},
			Result: &TestResult{
				sr: &SearchResponse{
					Users: []User{
						User{
							Id:     0,
							Name:   "Boyd Wolf",
							Age:    22,
							Gender: "male",
							About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
						},
					},
					NextPage: true,
				},
				err: nil,
			},
		},
	}

	// create TestServer
	ts := httptest.NewServer(http.HandlerFunc(handlerTestServer))

	// execute each TestCase
	for caseNum, tc := range cases {
		// create SearchClient
		sc := &SearchClient{
			AccessToken: tc.Input.sc.AccessToken,
			URL:         ts.URL,
		}

		// execute FindUsers
		sr, err := sc.FindUsers(*tc.Input.sr)

		// unexpected error
		if err != nil && tc.Result.err == nil {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
			continue
		}

		// check result if there is no error
		if !reflect.DeepEqual(tc.Result.sr, sr) {
			t.Errorf("[%d] wrong result, expected %#v, \n got %#v", caseNum, tc.Result.sr, sr)
			continue
		}
	}
	ts.Close()
}

func TestSearchClientWithFakeServer(t *testing.T) {

	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{
				URL: "bad_link",
			},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("unknown error Get bad_link?limit=0&offset=0&order_by=0&order_field=&query=: unsupported protocol scheme \"\""),
		},
	}

	_, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil {
		// if !strings.Contains(err.Error(), tc.Result.err.Error()) {
		if !reflect.DeepEqual(tc.Result.err, err) {

			t.Errorf("[%d] wrong error, expected %#v, got %#v", 0, tc.Result.err, err)
		}
	}
}

func TestSearchClientUnauthorized(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("Bad AccessToken"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}
func TestSearchClientTimeout(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("timeout for limit=0&offset=0&order_by=0&order_field=&query="),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 2)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalServerError(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("SearchServer fatal error"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalLimitError(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{
				Limit: -1,
			},
		},
		Result: &TestResult{
			err: fmt.Errorf("limit must be > 0"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalOffsetError(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{
				Offset: -1,
			},
		},
		Result: &TestResult{
			err: fmt.Errorf("offset must be > 0"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalBadJSON(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("cant unpack result json: invalid character '{' after top-level value"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmpJSON, _ := json.Marshal(struct{}{})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(tmpJSON)
		w.Write(tmpJSON)
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalOrderFieldError(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{
				OrderField: "About",
			},
		},
		Result: &TestResult{
			err: fmt.Errorf("OrderField About invalid"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orderFieldr := r.URL.Query()["order_field"][0]
		sr := SearchRequest{
			OrderField: orderFieldr,
		}
		if !(sr.OrderField == "Id" || sr.OrderField == "Age" || sr.OrderField == "Name") {
			ser := SearchErrorResponse{
				Error: "ErrorBadOrderField",
			}
			errJSON, _ := json.Marshal(ser)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(errJSON)
		}
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalStatusBadRequestBadJSON(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("cant unpack error json: invalid character '{' after top-level value"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmpJSON, _ := json.Marshal(struct{}{})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(tmpJSON)
		w.Write(tmpJSON)
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err == nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}

func TestSearchClientStatusInternalUnknowBadRequestError(t *testing.T) {
	tc := TestCase{
		Input: &TestInput{
			sc: &SearchClient{},
			sr: &SearchRequest{},
		},
		Result: &TestResult{
			err: fmt.Errorf("unknown bad request error: UnknowBadRequestError"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ser := SearchErrorResponse{
			Error: "UnknowBadRequestError",
		}
		errJSON, _ := json.Marshal(ser)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errJSON)
	}))
	defer ts.Close()

	tc.Input.sc.URL = ts.URL

	sr, err := tc.Input.sc.FindUsers(*tc.Input.sr)

	if err != nil && tc.Result.err != nil && sr == nil {
		if !reflect.DeepEqual(tc.Result.err, err) {
			// if err != tc.Result.err {
			t.Errorf("Wrong error, expected %#v, got %#v", tc.Result.err, err)
		}
	}
}
