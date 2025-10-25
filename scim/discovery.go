package scim

// ServiceProviderConfig represents the SCIM service provider configuration
type ServiceProviderConfig struct {
	Schemas               []string               `json:"schemas"`
	DocumentationURI      string                 `json:"documentationUri,omitempty"`
	Patch                 SupportedFeature       `json:"patch"`
	Bulk                  BulkFeature            `json:"bulk"`
	Filter                FilterFeature          `json:"filter"`
	ChangePassword        SupportedFeature       `json:"changePassword"`
	Sort                  SupportedFeature       `json:"sort"`
	Etag                  SupportedFeature       `json:"etag"`
	AuthenticationSchemes []AuthenticationScheme `json:"authenticationSchemes"`
}

// SupportedFeature indicates if a feature is supported
type SupportedFeature struct {
	Supported bool `json:"supported"`
}

// BulkFeature describes bulk operation capabilities
type BulkFeature struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

// FilterFeature describes filter capabilities
type FilterFeature struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

// AuthenticationScheme describes an authentication scheme
type AuthenticationScheme struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	SpecURI          string `json:"specUri,omitempty"`
	DocumentationURI string `json:"documentationUri,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
}

// SchemaDefinition represents a SCIM schema definition
type SchemaDefinition struct {
	ID          string                `json:"id"`
	Name        string                `json:"name,omitempty"`
	Description string                `json:"description,omitempty"`
	Attributes  []AttributeDefinition `json:"attributes,omitempty"`
}

// AttributeDefinition describes a SCIM attribute
type AttributeDefinition struct {
	Name            string                `json:"name"`
	Type            string                `json:"type"`
	SubAttributes   []AttributeDefinition `json:"subAttributes,omitempty"`
	MultiValued     bool                  `json:"multiValued"`
	Description     string                `json:"description,omitempty"`
	Required        bool                  `json:"required"`
	CaseExact       bool                  `json:"caseExact"`
	Mutability      string                `json:"mutability"`
	Returned        string                `json:"returned"`
	Uniqueness      string                `json:"uniqueness"`
	ReferenceTypes  []string              `json:"referenceTypes,omitempty"`
	CanonicalValues []string              `json:"canonicalValues,omitempty"`
}

// ResourceTypeDefinition represents a resource type
type ResourceTypeDefinition struct {
	Schemas          []string             `json:"schemas"`
	ID               string               `json:"id"`
	Name             string               `json:"name,omitempty"`
	Endpoint         string               `json:"endpoint"`
	Description      string               `json:"description,omitempty"`
	Schema           string               `json:"schema"`
	SchemaExtensions []SchemaExtensionRef `json:"schemaExtensions,omitempty"`
}

// SchemaExtensionRef references a schema extension
type SchemaExtensionRef struct {
	Schema   string `json:"schema"`
	Required bool   `json:"required"`
}

// GetServiceProviderConfig returns the service provider configuration
func GetServiceProviderConfig(authSchemes []AuthenticationScheme) *ServiceProviderConfig {
	if len(authSchemes) == 0 {
		authSchemes = []AuthenticationScheme{
			{
				Type:             "httpbasic",
				Name:             "HTTP Basic",
				Description:      "Authentication scheme using the HTTP Basic Standard",
				SpecURI:          "http://www.rfc-editor.org/info/rfc2617",
				DocumentationURI: "http://tools.ietf.org/html/rfc2617",
				Primary:          true,
			},
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication scheme using the OAuth Bearer Token Standard",
				SpecURI:     "http://www.rfc-editor.org/info/rfc6750",
			},
		}
	}

	return &ServiceProviderConfig{
		Schemas:          []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		DocumentationURI: "https://github.com/marcelom97/scimgateway",
		Patch: SupportedFeature{
			Supported: true,
		},
		Bulk: BulkFeature{
			Supported:      true,
			MaxOperations:  1000,
			MaxPayloadSize: 1048576, // 1MB
		},
		Filter: FilterFeature{
			Supported:  true,
			MaxResults: 1000,
		},
		ChangePassword: SupportedFeature{
			Supported: true,
		},
		Sort: SupportedFeature{
			Supported: true,
		},
		Etag: SupportedFeature{
			Supported: true,
		},
		AuthenticationSchemes: authSchemes,
	}
}

// GetUserSchema returns the User schema definition
func GetUserSchema() *SchemaDefinition {
	return &SchemaDefinition{
		ID:          SchemaUser,
		Name:        "User",
		Description: "User Account",
		Attributes: []AttributeDefinition{
			{
				Name:        "userName",
				Type:        "string",
				MultiValued: false,
				Required:    true,
				CaseExact:   false,
				Mutability:  "readWrite",
				Returned:    "default",
				Uniqueness:  "server",
			},
			{
				Name:        "name",
				Type:        "complex",
				MultiValued: false,
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
				SubAttributes: []AttributeDefinition{
					{Name: "formatted", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "familyName", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "givenName", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "middleName", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "honorificPrefix", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "honorificSuffix", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
				},
			},
			{
				Name:        "displayName",
				Type:        "string",
				MultiValued: false,
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
			},
			{
				Name:        "emails",
				Type:        "complex",
				MultiValued: true,
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
				SubAttributes: []AttributeDefinition{
					{Name: "value", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "display", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "type", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default", CanonicalValues: []string{"work", "home", "other"}},
					{Name: "primary", Type: "boolean", MultiValued: false, Mutability: "readWrite", Returned: "default"},
				},
			},
			{
				Name:        "active",
				Type:        "boolean",
				MultiValued: false,
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
			},
		},
	}
}

// GetGroupSchema returns the Group schema definition
func GetGroupSchema() *SchemaDefinition {
	return &SchemaDefinition{
		ID:          SchemaGroup,
		Name:        "Group",
		Description: "Group",
		Attributes: []AttributeDefinition{
			{
				Name:        "displayName",
				Type:        "string",
				MultiValued: false,
				Required:    true,
				CaseExact:   false,
				Mutability:  "readWrite",
				Returned:    "default",
				Uniqueness:  "none",
			},
			{
				Name:        "members",
				Type:        "complex",
				MultiValued: true,
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
				SubAttributes: []AttributeDefinition{
					{Name: "value", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default"},
					{Name: "$ref", Type: "reference", MultiValued: false, Mutability: "readWrite", Returned: "default", ReferenceTypes: []string{"User", "Group"}},
					{Name: "type", Type: "string", MultiValued: false, Mutability: "readWrite", Returned: "default", CanonicalValues: []string{"User", "Group"}},
				},
			},
		},
	}
}

// GetResourceTypes returns all resource type definitions
func GetResourceTypes() []ResourceTypeDefinition {
	return []ResourceTypeDefinition{
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			ID:          "User",
			Name:        "User",
			Endpoint:    "/Users",
			Description: "User Account",
			Schema:      SchemaUser,
			SchemaExtensions: []SchemaExtensionRef{
				{
					Schema:   "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
					Required: false,
				},
			},
		},
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			ID:          "Group",
			Name:        "Group",
			Endpoint:    "/Groups",
			Description: "Group",
			Schema:      SchemaGroup,
		},
	}
}
