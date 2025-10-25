package scim

import (
	"testing"

	"github.com/google/uuid"
)

func TestApplyResourceFilter(t *testing.T) {
	users := []*User{
		{ID: uuid.New().String(), UserName: "john.doe", Active: Bool(true)},
		{ID: uuid.New().String(), UserName: "jane.doe", Active: Bool(false)},
		{ID: uuid.New().String(), UserName: "bob.smith", Active: Bool(true)},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
		wantErr  bool
	}{
		{
			name:     "filter active users",
			filter:   "active eq true",
			expected: 2,
			wantErr:  false,
		},
		{
			name:     "filter by username",
			filter:   `userName eq "john.doe"`,
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "filter no match",
			filter:   `userName eq "nonexistent"`,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "empty filter returns all",
			filter:   "",
			expected: 3,
			wantErr:  false,
		},
		{
			name:     "invalid filter",
			filter:   "invalid syntax here",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyResourceFilter(users, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyResourceFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.expected {
				t.Errorf("ApplyResourceFilter() returned %d results, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestApplyResourcePagination(t *testing.T) {
	users := []*User{
		{ID: uuid.New().String(), UserName: "user1"},
		{ID: uuid.New().String(), UserName: "user2"},
		{ID: uuid.New().String(), UserName: "user3"},
		{ID: uuid.New().String(), UserName: "user4"},
		{ID: uuid.New().String(), UserName: "user5"},
	}

	tests := []struct {
		name             string
		startIndex       int
		count            int
		expectedLen      int
		expectedStart    int
		expectedItemsPer int
	}{
		{
			name:             "first page",
			startIndex:       1,
			count:            2,
			expectedLen:      2,
			expectedStart:    1,
			expectedItemsPer: 2,
		},
		{
			name:             "second page",
			startIndex:       3,
			count:            2,
			expectedLen:      2,
			expectedStart:    3,
			expectedItemsPer: 2,
		},
		{
			name:             "partial page",
			startIndex:       4,
			count:            10,
			expectedLen:      2,
			expectedStart:    4,
			expectedItemsPer: 2,
		},
		{
			name:             "count zero returns all",
			startIndex:       1,
			count:            0,
			expectedLen:      5,
			expectedStart:    1,
			expectedItemsPer: 5,
		},
		{
			name:             "negative count returns all",
			startIndex:       1,
			count:            -1,
			expectedLen:      5,
			expectedStart:    1,
			expectedItemsPer: 5,
		},
		{
			name:             "startIndex zero defaults to 1",
			startIndex:       0,
			count:            2,
			expectedLen:      2,
			expectedStart:    1,
			expectedItemsPer: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, startIndex, itemsPerPage := ApplyResourcePagination(users, tt.startIndex, tt.count)
			if len(result) != tt.expectedLen {
				t.Errorf("ApplyResourcePagination() returned %d items, expected %d", len(result), tt.expectedLen)
			}
			if startIndex != tt.expectedStart {
				t.Errorf("ApplyResourcePagination() startIndex = %d, expected %d", startIndex, tt.expectedStart)
			}
			if itemsPerPage != tt.expectedItemsPer {
				t.Errorf("ApplyResourcePagination() itemsPerPage = %d, expected %d", itemsPerPage, tt.expectedItemsPer)
			}
		})
	}
}

func TestApplyAttributeSelection(t *testing.T) {
	users := []*User{
		{
			ID:       "1",
			UserName: "john.doe",
			Active:   Bool(true),
			Name: &Name{
				GivenName:  "John",
				FamilyName: "Doe",
			},
			Emails: []Email{
				{Value: "john@example.com", Type: "work"},
			},
		},
	}

	tests := []struct {
		name         string
		attributes   []string
		excludedAttr []string
		checkFunc    func(*User) error
	}{
		{
			name:       "no selection returns all",
			attributes: nil,
			checkFunc: func(u *User) error {
				if u.UserName != "john.doe" {
					t.Error("userName should be present")
				}
				if u.Active == nil || !*u.Active {
					t.Error("active should be present and true")
				}
				if u.Name == nil {
					t.Error("name should be present")
				}
				return nil
			},
		},
		{
			name:       "select userName only",
			attributes: []string{"userName"},
			checkFunc: func(u *User) error {
				if u.UserName != "john.doe" {
					t.Error("userName should be present")
				}
				if u.Name != nil {
					t.Error("name should not be present")
				}
				return nil
			},
		},
		{
			name:         "exclude name",
			excludedAttr: []string{"name"},
			checkFunc: func(u *User) error {
				if u.UserName != "john.doe" {
					t.Error("userName should be present")
				}
				if u.Name != nil {
					t.Error("name should be excluded")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyAttributeSelection(users, tt.attributes, tt.excludedAttr)
			if err != nil {
				t.Errorf("ApplyAttributeSelection() error = %v", err)
				return
			}
			if len(result) != 1 {
				t.Errorf("ApplyAttributeSelection() returned %d results, expected 1", len(result))
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(result[0])
			}
		})
	}
}

func TestProcessListQuery(t *testing.T) {
	users := []*User{
		{ID: uuid.New().String(), UserName: "john.doe", Active: Bool(true)},
		{ID: uuid.New().String(), UserName: "jane.doe", Active: Bool(false)},
		{ID: uuid.New().String(), UserName: "bob.smith", Active: Bool(true)},
		{ID: uuid.New().String(), UserName: "alice.jones", Active: Bool(true)},
	}

	tests := []struct {
		name          string
		params        QueryParams
		expectedTotal int
		expectedItems int
		wantErr       bool
	}{
		{
			name: "filter and paginate",
			params: QueryParams{
				Filter:     "active eq true",
				StartIndex: 1,
				Count:      2,
			},
			expectedTotal: 3,
			expectedItems: 2,
			wantErr:       false,
		},
		{
			name: "filter only",
			params: QueryParams{
				Filter: `userName eq "john.doe"`,
			},
			expectedTotal: 1,
			expectedItems: 1,
			wantErr:       false,
		},
		{
			name: "paginate only",
			params: QueryParams{
				StartIndex: 2,
				Count:      2,
			},
			expectedTotal: 4,
			expectedItems: 2,
			wantErr:       false,
		},
		{
			name: "attribute selection",
			params: QueryParams{
				Attributes: []string{"userName"},
			},
			expectedTotal: 4,
			expectedItems: 4,
			wantErr:       false,
		},
		{
			name:          "no params returns all",
			params:        QueryParams{},
			expectedTotal: 4,
			expectedItems: 4,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessListQuery(users, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessListQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.TotalResults != tt.expectedTotal {
					t.Errorf("ProcessListQuery() TotalResults = %d, expected %d", result.TotalResults, tt.expectedTotal)
				}
				if len(result.Resources) != tt.expectedItems {
					t.Errorf("ProcessListQuery() returned %d items, expected %d", len(result.Resources), tt.expectedItems)
				}
				if result.ItemsPerPage != tt.expectedItems {
					t.Errorf("ProcessListQuery() ItemsPerPage = %d, expected %d", result.ItemsPerPage, tt.expectedItems)
				}
			}
		})
	}
}

func TestProcessListQueryWithGroups(t *testing.T) {
	groups := []*Group{
		{ID: "1", DisplayName: "Admins"},
		{ID: "2", DisplayName: "Users"},
		{ID: "3", DisplayName: "Developers"},
	}

	params := QueryParams{
		Filter:     `displayName co "Dev"`,
		StartIndex: 1,
		Count:      10,
	}

	result, err := ProcessListQuery(groups, params)
	if err != nil {
		t.Errorf("ProcessListQuery() error = %v", err)
		return
	}

	if result.TotalResults != 1 {
		t.Errorf("ProcessListQuery() TotalResults = %d, expected 1", result.TotalResults)
	}

	if len(result.Resources) != 1 {
		t.Errorf("ProcessListQuery() returned %d items, expected 1", len(result.Resources))
	}

	if result.Resources[0].DisplayName != "Developers" {
		t.Errorf("ProcessListQuery() returned wrong group: %s", result.Resources[0].DisplayName)
	}
}
