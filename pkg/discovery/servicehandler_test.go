package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

var exchangeFile = anypoint.ExchangeFile{
	Classifier:  "fat-oas",
	DownloadURL: "abc.com",
}

var exchangeAsset = anypoint.ExchangeAsset{
	AssetID:      "petstore-3",
	AssetType:    "rest-api",
	Categories:   nil,
	CreatedAt:    time.Now(),
	Description:  "",
	Files:        []anypoint.ExchangeFile{exchangeFile},
	GroupID:      "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4",
	Icon:         "",
	ID:           "d3ada710-fc7b-4fc7-b8b9-4ccfc0f872e4/petstore-3/1.0.0",
	Instances:    nil,
	Labels:       nil,
	MinorVersion: "1.0",
	ModifiedAt:   time.Time{},
	Name:         "petstore-3",
	Public:       false,
	Snapshopt:    false,
	Status:       "published",
	Version:      "1.0.0",
	VersionGroup: "v1",
}

func TestServiceHandler(t *testing.T) {
	stage := "Sandbox"
	content := `{"openapi":"3.0.1","servers":[{"url":"https://abc.com"}], "paths":{}, "info":{"title":"petstore3"}}`
	policies := anypoint.Policies{Policies: []anypoint.Policy{
		{
			Template: anypoint.Template{
				AssetId: anypoint.ClientID,
			},
		},
	}}
	mc := &anypoint.MockAnypointClient{}
	mc.On("GetPolicies").Return(policies, nil)
	mc.On("GetExchangeAsset").Return(&exchangeAsset, nil)
	mc.On("GetExchangeFileContent").Return([]byte(content), nil)
	mc.On("GetExchangeAssetIcon").Return("", "", nil)
	sh := &serviceHandler{
		assetCache:          cache.New(),
		freshCache:          cache.New(),
		stage:               stage,
		discoveryTags:       []string{"tag1"},
		discoveryIgnoreTags: []string{"nah"},
		client:              mc,
	}
	details := sh.ToServiceDetails(&asset)
	assert.Equal(t, 1, len(details))
	item := details[0]
	assert.Equal(t, asset.APIs[0].AssetID, item.APIName)
	assert.Equal(t, "verify-api-key", item.AuthPolicy)
	assert.Equal(t, fmt.Sprint(asset.APIs[0].ID), item.ID)
	assert.Equal(t, apic.Oas3, item.ResourceType)
	assert.Equal(t, stage, item.Stage)
	assert.Equal(t, asset.ExchangeAssetName, item.Title)
	assert.Equal(t, asset.APIs[0].AssetVersion, item.Version)
	assert.Equal(t, asset.APIs[0].Tags, item.Tags)
	assert.NotEmpty(t, item.ServiceAttributes["checksum"])
}

func TestServiceHandlerDidNotDiscoverAPI(t *testing.T) {
	stage := "Sandbox"
	policies := anypoint.Policies{Policies: []anypoint.Policy{
		{
			Template: anypoint.Template{
				AssetId: anypoint.ClientID,
			},
		},
	}}
	mc := &anypoint.MockAnypointClient{}
	mc.On("GetPolicies").Return(policies, nil)
	sh := &serviceHandler{
		assetCache:          cache.New(),
		freshCache:          cache.New(),
		stage:               stage,
		discoveryTags:       []string{"nothing"},
		discoveryIgnoreTags: []string{"nah"},
		client:              mc,
	}
	details := sh.ToServiceDetails(&asset)
	assert.Equal(t, 0, len(details))
	assert.Equal(t, 0, len(mc.Calls))
}

func TestServiceHandlerGetPolicyError(t *testing.T) {
	stage := "Sandbox"
	policies := anypoint.Policies{Policies: []anypoint.Policy{}}
	mc := &anypoint.MockAnypointClient{}
	expectedErr := fmt.Errorf("failed to get policies")
	mc.On("GetPolicies").Return(policies, expectedErr)
	sh := &serviceHandler{
		assetCache:          cache.New(),
		freshCache:          cache.New(),
		stage:               stage,
		discoveryTags:       []string{},
		discoveryIgnoreTags: []string{},
		client:              mc,
	}
	sd, err := sh.getServiceDetail(&asset, &asset.APIs[0])

	assert.Nil(t, sd)
	assert.Equal(t, expectedErr, err)
}

func TestServiceHandlerGetExchangeAssetError(t *testing.T) {
	stage := "Sandbox"
	policies := anypoint.Policies{Policies: []anypoint.Policy{}}
	mc := &anypoint.MockAnypointClient{}
	expectedErr := fmt.Errorf("failed to get exchange asset")
	mc.On("GetPolicies").Return(policies, nil)
	mc.On("GetExchangeAsset").Return(&anypoint.ExchangeAsset{}, expectedErr)
	sh := &serviceHandler{
		assetCache:          cache.New(),
		freshCache:          cache.New(),
		stage:               stage,
		discoveryTags:       []string{},
		discoveryIgnoreTags: []string{},
		client:              mc,
	}
	sd, err := sh.getServiceDetail(&asset, &asset.APIs[0])

	assert.Nil(t, sd)
	assert.Equal(t, expectedErr, err)
}

func TestShouldDiscoverAPIBasedOnTags(t *testing.T) {
	tests := []struct {
		name          string
		discoveryTags []string
		ignoreTags    []string
		apiTags       []string
		expected      bool
		endpoint      string
	}{
		{
			name:          "Should discover if matching discovery tag exists on API",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{},
			apiTags:       []string{"discover"},
			expected:      true,
			endpoint:      "abc.com",
		},
		{
			name:          "Should not discover if API has a tag to be ignored",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{"donotdiscover"},
			apiTags:       []string{"donotdiscover"},
			expected:      false,
			endpoint:      "abc.com",
		},
		{
			name:          "Should not discover if API does not have any tags that the agent's config has",
			ignoreTags:    []string{"donotdiscover"},
			discoveryTags: []string{"discover"},
			apiTags:       []string{},
			expected:      false,
			endpoint:      "abc.com",
		},
		{
			name:          "Should discover if API as well as agent's config have no discovery tags",
			discoveryTags: []string{},
			ignoreTags:    []string{},
			apiTags:       []string{},
			expected:      true,
			endpoint:      "abc.com",
		},
		{
			name:          "Should not discover if API has both - a tag to be discovered and a tag to be ignored",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{"donotdiscover"},
			apiTags:       []string{"discover", "donotdiscover"},
			expected:      false,
			endpoint:      "abc.com",
		},
		{
			name:          "Should not discover if the endpoint is empty",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{"donotdiscover"},
			apiTags:       []string{"discover"},
			expected:      false,
			endpoint:      "",
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			ok := shouldDiscoverAPI(tc.endpoint, tc.discoveryTags, tc.ignoreTags, tc.apiTags)
			assert.Equal(t, tc.expected, ok)
		})
	}
}

func TestGetExchangeAssetSpecFile(t *testing.T) {
	tests := []struct {
		name     string
		files    []anypoint.ExchangeFile
		expected *anypoint.ExchangeFile
	}{
		{
			name:     "Should return nil if the Exchange asset has no files",
			files:    nil,
			expected: nil,
		},
		{
			name: "Should return nil if the Exchange asset has a file that is not of expected classifier",
			files: []anypoint.ExchangeFile{
				{Classifier: "oas3"},
			},
			expected: nil,
		},
		{
			name: "Should return the OAS asset, since it is an expected classifier",
			files: []anypoint.ExchangeFile{
				{Classifier: "oas"},
			},
			expected: &anypoint.ExchangeFile{
				Classifier: "oas",
			},
		},
		{
			name: "Should return the fat-oas asset, since it is an expected classifier",
			files: []anypoint.ExchangeFile{
				{Classifier: "fat-oas"},
			},
			expected: &anypoint.ExchangeFile{
				Classifier: "fat-oas",
			},
		},
		{
			name: "Should return the fat-oas asset, since it is an expected classifier",
			files: []anypoint.ExchangeFile{
				{Classifier: "wsdl"},
			},
			expected: &anypoint.ExchangeFile{
				Classifier: "wsdl",
			},
		},
		{
			name: "Should sort files, and return the first matching classifier",
			files: []anypoint.ExchangeFile{
				{Classifier: "wsdl"},
				{Classifier: "oas"},
			},
			expected: &anypoint.ExchangeFile{
				Classifier: "oas",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sd := getExchangeAssetSpecFile(tc.files)
			assert.Equal(t, tc.expected, sd)
		})
	}
}

func Test_checksum(t *testing.T) {
	s1 := checksum(&asset, apic.Passthrough)
	s2 := checksum(&asset, anypoint.ClientID)
	assert.NotEmpty(t, s1)
	assert.NotEqual(t, s1, s2)
}

func Test_getAuthPolicy(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		policies anypoint.Policies
	}{
		{
			name:     "should return the policy as APIKey when the mulesoft policy is client-id-enforcement",
			expected: apic.Apikey,
			policies: anypoint.Policies{
				Policies: []anypoint.Policy{
					{
						Template: anypoint.Template{
							AssetId: anypoint.ClientID,
						},
					},
				},
			},
		},
		{
			name:     "should return the policy as OAuth when the mulesoft policy is oauth",
			expected: apic.Oauth,
			policies: anypoint.Policies{
				Policies: []anypoint.Policy{
					{
						Template: anypoint.Template{
							AssetId: anypoint.ExternalOauth,
						},
					},
				},
			},
		},
		{
			name:     "should return the first policy that matches 'client-id-enforcement'",
			expected: apic.Apikey,
			policies: anypoint.Policies{
				Policies: []anypoint.Policy{
					{
						Template: anypoint.Template{
							AssetId: "fake",
						},
					},
					{
						Template: anypoint.Template{
							AssetId: anypoint.ClientID,
						},
					},
				},
			},
		},
		{
			name:     "should return the policy as pass-through when there are no policies in the array",
			expected: apic.Passthrough,
			policies: anypoint.Policies{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy := getAuthPolicy(tc.policies)
			assert.Equal(t, policy, tc.expected)
		})
	}
}

func Test_getSpecType(t *testing.T) {
	tests := []struct {
		name         string
		file         *anypoint.ExchangeFile
		specContent  []byte
		expectedType string
		expectedErr  error
	}{
		{
			name: "should return the spec type as WSDL",
			file: &anypoint.ExchangeFile{
				Classifier: apic.Wsdl,
			},
			specContent:  []byte(""),
			expectedType: apic.Wsdl,
		},
		{
			name: "should return the spec type as OAS2",
			file: &anypoint.ExchangeFile{
				Classifier: apic.Oas2,
			},
			specContent:  []byte(`{"basePath":"google.com","host":"","schemes":[""],"swagger":"2.0"}`),
			expectedType: apic.Oas2,
		},
		{
			name: "should return the spec type as OAS3",
			file: &anypoint.ExchangeFile{
				Classifier: apic.Oas3,
			},
			specContent:  []byte(`{"openapi": "3.0.1"}`),
			expectedType: apic.Oas3,
		},
		{
			name: "should return the specType as an empty string when the specContent is nil",
			file: &anypoint.ExchangeFile{
				Classifier: apic.Oas3,
			},
			specContent:  nil,
			expectedType: "",
		},
		{
			name: "should return an error when given an invalid spec",
			file: &anypoint.ExchangeFile{
				Classifier: apic.Oas3,
			},
			specContent:  []byte("abc"),
			expectedType: "",
			expectedErr:  fmt.Errorf("error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			specType, err := getSpecType(tc.file, tc.specContent)
			if tc.expectedErr != nil {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.expectedType, specType)
		})
	}
}

func Test_updateSpec(t *testing.T) {
	tests := []struct {
		name            string
		specType        string
		endpoint        string
		content         []byte
		expectedContent []byte
		authPolicy      string
	}{
		{
			name:            "should update an OAS 2 spec with APIKey security",
			specType:        apic.Oas2,
			endpoint:        "https://abc.com/v1",
			content:         []byte(`{"basePath": "/v2","host": "oldhost.com","schemes": ["http"],"swagger": "2.0","info": {"title": "petstore2"},"paths": {}}`),
			expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","info":{"title":"petstore2","version":""},"schemes":["http"],"securityDefinitions":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e","in":"header","name":"Authorization","type":"apiKey"}},"swagger":"2.0"}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:            "should update an OAS 2 spec with OAuth security",
			specType:        apic.Oas2,
			endpoint:        "https://abc.com/v1",
			content:         []byte(`{"basePath":"/v2","host":"oldhost.com","schemes":["http"],"swagger":"2.0","info":{"title":"petstore2"},"paths":{}}`),
			expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","info":{"title":"petstore2","version":""},"schemes":["http"],"securityDefinitions":{"oauth":{"authorizationUrl":"dummy.io","flow":"implicit","type":"oauth2"}},"swagger":"2.0"}`),
			authPolicy:      apic.Oauth,
		},
		{
			name:            "should update an OAS 3 spec with OAuth security",
			specType:        apic.Oas3,
			endpoint:        "https://abc.com",
			content:         []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}],"paths":{},"info":{"title":"petstore3"}}`),
			expectedContent: []byte(`{"components":{"securitySchemes":{"Oauth":{"description":"This API uses OAuth 2 with the implicit grant flow","flows":{"implicit":{"authorizationUrl":"dummy.io","scopes":{}}},"type":"oauth2"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":{},"servers":[{"url":"google.com"}]}`),
			authPolicy:      apic.Oauth,
		},
		{
			name:            "should update an OAS 3 spec with APIKey security",
			specType:        apic.Oas3,
			endpoint:        "https://abc.com",
			content:         []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}],"paths":{},"info":{"title":"petstore3"}}`),
			expectedContent: []byte(`{"components":{"securitySchemes":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e","in":"header","name":"Authorization","type":"apiKey"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":{},"servers":[{"url":"google.com"}]}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:            "should update a WSDL spec",
			specType:        apic.Wsdl,
			endpoint:        "https://abc.com",
			content:         []byte(""),
			expectedContent: []byte(""),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := updateSpec(tc.specType, tc.endpoint, tc.authPolicy, tc.content)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedContent, content)
		})
	}
}
