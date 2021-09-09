package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Users struct {
	Users []User `xml:"row"`
}

type By func(u1, u2 *User) bool

func (by By) Sort(users []User) {
	us := &userSorter{
		users: users,
		by:    by,
	}
	sort.Sort(us)
}

type userSorter struct {
	users []User
	by    By
}

func (s *userSorter) Len() int {
	return len(s.users)
}

func (s *userSorter) Swap(i, j int) {
	s.users[i], s.users[j] = s.users[j], s.users[i]
}

func (s *userSorter) Less(i, j int) bool {
	return s.by(&s.users[i], &s.users[j])
}

type TestCase struct {
	Request       SearchRequest
	Response      *SearchResponse
	ErrorResponse SearchErrorResponse
	IsErr         bool
	Status        int
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil || orderBy < -1 || orderBy > 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "ErrorBadOrderBy"}`))
		return
	}
	orderField := strings.ToLower(r.FormValue("order_field"))
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil || limit < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "ErrorBadLimit"}`))
		return
	}
	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil || offset < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "ErrorBadOffset"}`))
		return
	}
	users := getUsers()
	if orderBy != 0 {
		var by By
		switch orderField {
		case "id":
			by = func(u1, u2 *User) bool {
				if orderBy == 1 {
					return u1.Id < u2.Id
				}
				return u1.Id > u2.Id
			}
		case "name":
			by = func(u1, u2 *User) bool {
				if orderBy == 1 {
					return u1.Name < u2.Name
				}
				return u1.Name > u2.Name
			}
		case "age":
			by = func(u1, u2 *User) bool {
				if orderBy == 1 {
					return u1.Age < u2.Age
				}
				return u1.Age > u2.Age
			}
		case "about":
			by = func(u1, u2 *User) bool {
				if orderBy == 1 {
					return u1.About < u2.About
				}
				return u1.About > u2.About
			}
		case "gender":
			by = func(u1, u2 *User) bool {
				if orderBy == 1 {
					return u1.Gender < u2.Gender
				}
				return u1.Gender > u2.Gender
			}
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "ErrorBadOrderField"}`))
			return
		}
		By(by).Sort(users)
	}
	if offset >= len(users) {
		offset = len(users) - 1
	}
	if offset+limit > len(users) {
		limit = len(users) - offset
	}
	cropped := users[offset : offset+limit]
	data, err := json.Marshal(cropped)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "ErrorInternalServerError"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func TestTestCases(t *testing.T) {
	cases := []TestCase{
		{
			Request:  SearchRequest{Limit: -1},
			IsErr:    true,
			Response: nil,
		},
		{
			Request:  SearchRequest{Offset: -1},
			IsErr:    true,
			Response: nil,
		},
		{
			Request:  SearchRequest{Offset: 1, Limit: 1, OrderBy: 2},
			IsErr:    true,
			Response: nil,
		},
		{
			Request:  SearchRequest{Offset: 1, Limit: 26, OrderBy: 1, OrderField: "test"},
			Response: nil,
			IsErr:    true,
		},
		{
			Request: SearchRequest{Offset: 0, Limit: 1},
			Response: &SearchResponse{Users: []User{
				{
					Id:     0,
					Name:   "Boyd",
					Age:    22,
					About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
					Gender: "male",
				},
			}, NextPage: true},
			IsErr: false,
		},
		{
			Request: SearchRequest{Offset: 34, Limit: 1, OrderField: "id", OrderBy: -1},
			Response: &SearchResponse{Users: []User{
				{
					Id:     0,
					Name:   "Boyd",
					Age:    22,
					About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
					Gender: "male",
				},
			}, NextPage: false},
			IsErr: false,
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for i, item := range cases {
		client := &SearchClient{
			AccessToken: "test",
			URL:         ts.URL,
		}
		res, err := client.FindUsers(item.Request)
		if err != nil && !item.IsErr {
			t.Errorf("[%d] Unexpected error: %#v", i, err)
		}
		if err == nil && item.IsErr {
			t.Errorf("[%d] Expected error, got nil", i)
		}
		if !reflect.DeepEqual(item.Response, res) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", i, item.Response, res)
		}
	}
}

func TestUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test",
		URL:         ts.URL,
	}
	req := &SearchRequest{Limit: 1}
	if _, err := client.FindUsers(*req); err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test",
		URL:         ts.URL,
	}
	req := &SearchRequest{Limit: 1}
	if _, err := client.FindUsers(*req); err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("test"))
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test",
		URL:         ts.URL,
	}
	req := &SearchRequest{Limit: 1}
	if _, err := client.FindUsers(*req); err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestWrongJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test",
		URL:         ts.URL,
	}
	req := &SearchRequest{Limit: 1}
	if _, err := client.FindUsers(*req); err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * 1001)
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test",
		URL:         ts.URL,
	}
	req := &SearchRequest{Limit: 1}
	if _, err := client.FindUsers(*req); err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestClientError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "")
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test",
		URL:         "",
	}
	req := &SearchRequest{Limit: 1}
	if _, err := client.FindUsers(*req); err == nil {
		t.Error("Expected error, got nil")
	}
}

const filePath string = "./dataset.xml"

func getUsers() []User {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	data := &Users{}
	xml.Unmarshal(fileContents, &data)
	return data.Users
}

func TestGetUsers(t *testing.T) {
	users := getUsers()
	if len(users) != 35 {
		t.Error("Count users should be 35")
	}
}
