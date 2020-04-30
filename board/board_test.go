package board

import (
	"testing"
)

type testBoards struct {
	input  string
	result string
}

type testSegments struct {
	boardURL     string
	pathSegments []string
}

func TestTrimPath(t *testing.T) {
	var testTrim = []testBoards{
		{"/user1/board/", "user1/board"},
		{"/user1/board", "user1/board"},
		{"user1/board/", "user1/board"},
		{"user1/board", "user1/board"},
		{"/user1/", "user1"},
		{"/user1", "user1"},
		{"user1/", "user1"},
		{"user1", "user1"},
	}

	for _, user := range testTrim {
		trimmedPath, _ := trimPath(user.input)
		if user.result != trimmedPath {
			t.Errorf("Got path: %s / Expected: %s ", trimmedPath, user.result)
		}
	}
}

func TestParseUserError(t *testing.T) {
	var testUserError = []testBoards{
		{"", "Parse user failed: Received empty path"},
	}

	for _, user := range testUserError {
		_, err := parseUser(user.input)
		if err.Error() != user.result {
			t.Errorf("Got error: %s / Expected error: %s ", err.Error(), user.result)
		}
	}
}

func TestParseUser(t *testing.T) {
	var testUser = []testBoards{
		{"/user1/board", "user1"},
		{"/user2/board/section1", "user2"},
	}

	for _, user := range testUser {
		parsedUser, _ := parseUser(user.input)
		if user.result != parsedUser {
			t.Errorf("Got user %s / Expected: %s ", parsedUser, user.result)
		}
	}
}

func TestParsePath(t *testing.T) {
	var testPath = []testBoards{
		{"/user1/board", "board"},
		{"/user2/board/section1", "board/section1"},
	}

	for _, path := range testPath {
		parsedPath, _ := parsePath(path.input)
		if path.result != parsedPath {
			t.Errorf("Got path: %s / Expected: %s", parsedPath, path.result)
		}
	}
}

func TestPathSegments(t *testing.T) {
	var testPathSegments = []testSegments{
		testSegments{"/board", []string{"board"}},
		testSegments{"/board/section1", []string{"board", "section1"}},
	}

	for _, tps := range testPathSegments {
		parsedPathSegments, _ := parsePathSegments(tps.boardURL)
		for x := 0; x < len(tps.pathSegments); x++ {
			if parsedPathSegments[x] != tps.pathSegments[x] {
				t.Errorf("Path segment '%s' doesn't match '%s'", tps.pathSegments[x], parsedPathSegments[x])
			}
		}
	}
}
