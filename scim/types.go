package scim

import (
	"encoding/json"
	"strings"
	"time"
)

// Resource represents a SCIM resource with common attributes
type Resource struct {
	ID         string         `json:"id"`
	ExternalID string         `json:"externalId,omitempty"`
	Meta       *Meta          `json:"meta,omitempty"`
	Schemas    []string       `json:"schemas"`
	Attributes map[string]any `json:"-"`
}

// Meta contains metadata about a SCIM resource
type Meta struct {
	ResourceType string     `json:"resourceType"`
	Created      *time.Time `json:"created,omitempty"`
	LastModified *time.Time `json:"lastModified,omitempty"`
	Location     string     `json:"location,omitempty"`
	Version      string     `json:"version,omitempty"`
}

// User represents a SCIM User resource
type User struct {
	ID               string            `json:"id"`
	ExternalID       string            `json:"externalId,omitempty"`
	Meta             *Meta             `json:"meta,omitempty"`
	Schemas          []string          `json:"schemas"`
	UserName         string            `json:"userName,omitempty"`
	Name             *Name             `json:"name,omitempty"`
	DisplayName      string            `json:"displayName,omitempty"`
	NickName         string            `json:"nickName,omitempty"`
	ProfileURL       string            `json:"profileUrl,omitempty"`
	Title            string            `json:"title,omitempty"`
	UserType         string            `json:"userType,omitempty"`
	PreferredLang    string            `json:"preferredLanguage,omitempty"`
	Locale           string            `json:"locale,omitempty"`
	Timezone         string            `json:"timezone,omitempty"`
	Active           *bool             `json:"active,omitempty"`
	Password         string            `json:"password,omitempty"`
	Emails           []Email           `json:"emails,omitempty"`
	PhoneNumbers     []PhoneNumber     `json:"phoneNumbers,omitempty"`
	IMs              []IM              `json:"ims,omitempty"`
	Photos           []Photo           `json:"photos,omitempty"`
	Addresses        []Address         `json:"addresses,omitempty"`
	Groups           []GroupRef        `json:"groups,omitempty"`
	Entitlements     []Entitlement     `json:"entitlements,omitempty"`
	Roles            []Role            `json:"roles,omitempty"`
	X509Certificates []X509Certificate `json:"x509Certificates,omitempty"`
	EnterpriseUser   map[string]any    `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
}

// Name represents a user's name components
type Name struct {
	Formatted       string `json:"formatted,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	MiddleName      string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

// MultiValuedAttribute represents a generic multi-valued SCIM attribute
type MultiValuedAttribute[T any] struct {
	Value   T       `json:"value"`
	Type    string  `json:"type,omitempty"`
	Primary Boolean `json:"primary,omitempty"`
	Display string  `json:"display,omitempty"`
}

type Boolean bool

func (b *Boolean) UnmarshalJSON(data []byte) error {
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	switch v := val.(type) {
	case bool:
		*b = Boolean(v)
		return nil
	case string:
		if strings.ToLower(v) == "true" {
			*b = Boolean(true)
		} else if strings.ToLower(v) == "false" {
			*b = Boolean(false)
		} else {
			return nil
		}
		return nil
	default:
		return nil
	}
}

func (b Boolean) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(b))
}

// Email represents an email address
type Email = MultiValuedAttribute[string]

// PhoneNumber represents a phone number
type PhoneNumber = MultiValuedAttribute[string]

// IM represents an instant messaging address
type IM = MultiValuedAttribute[string]

// Photo represents a photo URL
type Photo = MultiValuedAttribute[string]

// Address represents a physical mailing address
type Address struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"streetAddress,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	Country       string `json:"country,omitempty"`
	Type          string `json:"type,omitempty"`
	Primary       bool   `json:"primary,omitempty"`
}

// GroupRef represents a reference to a group
type GroupRef struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
}

// Entitlement represents an entitlement
type Entitlement = MultiValuedAttribute[string]

// Role represents a role
type Role = MultiValuedAttribute[string]

// X509Certificate represents an X.509 certificate
type X509Certificate = MultiValuedAttribute[string]

// Group represents a SCIM Group resource
type Group struct {
	ID          string      `json:"id"`
	ExternalID  string      `json:"externalId,omitempty"`
	Meta        *Meta       `json:"meta,omitempty"`
	Schemas     []string    `json:"schemas"`
	DisplayName string      `json:"displayName"`
	Members     []MemberRef `json:"members,omitempty"`
}

// MemberRef represents a reference to a group member
type MemberRef struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Display string `json:"display,omitempty"`
}

// ListResponse represents a SCIM list response with generic resource type
type ListResponse[T any] struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    []T      `json:"Resources"`
}

// Error represents a SCIM error response
type Error struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	Detail   string   `json:"detail,omitempty"`
	ScimType string   `json:"scimType,omitempty"`
}

// PatchOp represents a SCIM PATCH operation
type PatchOp struct {
	Schemas    []string         `json:"schemas"`
	Operations []PatchOperation `json:"Operations"`
}

// PatchOperation represents a single SCIM PATCH operation
type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

// QueryParams represents query parameters for list operations
type QueryParams struct {
	Filter       string
	Attributes   []string
	ExcludedAttr []string
	StartIndex   int
	Count        int
	SortBy       string
	SortOrder    string
}

// Bool returns a pointer to the given bool value
func Bool(b bool) *bool {
	return &b
}
