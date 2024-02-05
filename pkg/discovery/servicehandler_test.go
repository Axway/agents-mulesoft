package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agent-sdk/pkg/cache"

	corecfg "github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agent-sdk/pkg/agent"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Axway/agents-mulesoft/pkg/discovery/mocks"

	"github.com/Axway/agents-mulesoft/pkg/subscription"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/apic"

	"github.com/stretchr/testify/assert"

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
	Snapshot:     false,
	Status:       "published",
	Version:      "1.0.0",
	VersionGroup: "v1",
}

func TestServiceHandler(t *testing.T) {
	type testCase struct {
		content              string
		policies             []anypoint.Policy
		exchangeAsset        *anypoint.ExchangeAsset
		exchangeAssetIcon    string
		expectedResourceType string
		expectedTitle        string
		expectedVersion      string
	}
	cases := []testCase{
		{
			content: "#%RAML 1.0\ntitle: API with Examples\ndescription: Grand Theft Auto:Vice City\nversion: v3\nprotocols: [HTTP,HTTPS]\nbaseUri: https://na1.salesforce.com:4000/services/data/{version}/chatter",
			policies: []anypoint.Policy{
				{PolicyTemplateID: common.ClientIDEnforcement},
			},
			exchangeAsset:        &exchangeAsset,
			expectedResourceType: apic.Raml,
		},
		{
			content: `{"openapi":"3.0.1","servers":[{"url":"https://abc.com"}], "paths":{}, "info":{"title":"petstore3"}}`,
			policies: []anypoint.Policy{
				{PolicyTemplateID: common.ClientIDEnforcement},
			},
			exchangeAsset:        &exchangeAsset,
			expectedResourceType: apic.Oas3,
		},
	}
	for _, c := range cases {
		mc := &anypoint.MockAnypointClient{}
		mc.On("GetPolicies").Return(c.policies, nil)
		mc.On("GetExchangeAsset").Return(c.exchangeAsset, nil)
		mc.On("GetExchangeFileContent").Return([]byte(c.content), nil)
		mc.On("GetExchangeAssetIcon").Return("", "", nil)

		msh := &mockSchemaHandler{}
		sh := &serviceHandler{
			muleEnv:             "Sandbox",
			discoveryTags:       []string{"tag1"},
			discoveryIgnoreTags: []string{"nah"},
			client:              mc,
			schemas:             msh,
			cache:               cache.New(),
		}
		list := sh.ToServiceDetails(&asset)
		api := asset.APIs[0]
		assert.Equal(t, 1, len(list))
		item := list[0]

		assert.Equal(t, asset.APIs[0].AssetID, item.APIName)
		assert.Equal(t, apic.Apikey, item.AuthPolicy)
		assert.Equal(t, fmt.Sprint(asset.ID), item.ID)
		assert.Equal(t, c.expectedResourceType, item.ResourceType)
		assert.Equal(t, api.AssetVersion, item.Stage)
		assert.Equal(t, asset.ExchangeAssetName, item.Title)
		assert.Equal(t, api.AssetVersion, item.Version)
		assert.Equal(t, api.Tags, item.Tags)
		assert.NotEmpty(t, item.AgentDetails[common.AttrChecksum])
		assert.Equal(t, fmt.Sprint(api.ID), item.AgentDetails[common.AttrAPIID])
		assert.Equal(t, fmt.Sprint(asset.ID), item.AgentDetails[common.AttrAssetID])
		assert.Equal(t, api.AssetVersion, item.AgentDetails[common.AttrAssetVersion])
		assert.Equal(t, api.ProductVersion, item.AgentDetails[common.AttrProductVersion])

		// Should find the api in the cache
		cachedItem, err := sh.cache.Get(item.AgentDetails[common.AttrChecksum])
		assert.Nil(t, err)
		assert.Equal(t, api, cachedItem)

		// Should find the api in the cache by the secondary key
		cachedItem, err = sh.cache.GetBySecondaryKey(common.FormatAPICacheKey(fmt.Sprint(api.ID), api.ProductVersion))
		assert.Nil(t, err)
		assert.Equal(t, api, cachedItem)

		// Should not discover an API that is saved in the cache.
		list = sh.ToServiceDetails(&asset)
		assert.Equal(t, 0, len(list))
	}
}

func TestServiceHandlerSLAPolicy(t *testing.T) {
	cc := &mocks.MockCentralClient{}
	cc.On("RegisterSubscriptionSchema").Return(nil)
	agent.Initialize(&corecfg.CentralConfiguration{})
	agent.InitializeForTest(cc)
	content := `{"openapi":"3.0.1","servers":[{"url":"https://abc.com"}], "paths":{}, "info":{"title":"petstore3"}}`
	policies := []anypoint.Policy{
		{
			PolicyTemplateID: common.SLABased,
		},
	}
	mc := &anypoint.MockAnypointClient{}
	mc.On("GetPolicies").Return(policies, nil)
	mc.On("GetExchangeAsset").Return(&exchangeAsset, nil)
	mc.On("GetExchangeFileContent").Return([]byte(content), nil)
	mc.On("GetExchangeAssetIcon").Return("", "", nil)

	msh := &mockSchemaHandler{}
	sh := &serviceHandler{
		muleEnv:             "Sandbox",
		discoveryTags:       []string{"tag1"},
		discoveryIgnoreTags: []string{"nah"},
		client:              mc,
		schemas:             msh,
		cache:               cache.New(),
	}

	details := sh.ToServiceDetails(&asset)

	assert.Equal(t, 1, len(details))
	assert.Equal(t, fmt.Sprint(apiID), details[0].SubscriptionName)

}

func TestServiceHandlerDidNotDiscoverAPI(t *testing.T) {
	policies := []anypoint.Policy{
		{
			PolicyTemplateID: common.ClientIDEnforcement,
		},
	}
	mc := &anypoint.MockAnypointClient{}
	mc.On("GetPolicies").Return(policies, nil)
	sh := &serviceHandler{
		muleEnv:             "Sandbox",
		discoveryTags:       []string{"nothing"},
		discoveryIgnoreTags: []string{"nah"},
		client:              mc,
		cache:               cache.New(),
		schemas:             &mockSchemaHandler{},
	}
	details := sh.ToServiceDetails(&asset)
	assert.Equal(t, 0, len(details))
	assert.Equal(t, 0, len(mc.Calls))
}

func TestServiceHandlerGetPolicyError(t *testing.T) {
	stage := "Sandbox"
	policies := []anypoint.Policy{}
	mc := &anypoint.MockAnypointClient{}
	expectedErr := fmt.Errorf("failed to get policies")
	mc.On("GetPolicies").Return(policies, expectedErr)
	sh := &serviceHandler{
		muleEnv:             stage,
		discoveryTags:       []string{},
		discoveryIgnoreTags: []string{},
		client:              mc,
		cache:               cache.New(),
		schemas:             &mockSchemaHandler{},
	}
	sd, err := sh.getServiceDetail(&asset, &asset.APIs[0])

	assert.Nil(t, sd)
	assert.Equal(t, expectedErr, err)
}

func TestServiceHandlerGetExchangeAssetError(t *testing.T) {
	stage := "Sandbox"
	policies := []anypoint.Policy{}
	mc := &anypoint.MockAnypointClient{}
	expectedErr := fmt.Errorf("failed to get exchange asset")
	mc.On("GetPolicies").Return(policies, nil)
	mc.On("GetExchangeAsset").Return(&anypoint.ExchangeAsset{}, expectedErr)
	sh := &serviceHandler{
		muleEnv:             stage,
		discoveryTags:       []string{},
		discoveryIgnoreTags: []string{},
		client:              mc,
		schemas:             &mockSchemaHandler{},
		cache:               cache.New(),
	}
	sd, err := sh.getServiceDetail(&asset, &asset.APIs[0])

	assert.Nil(t, sd)
	assert.Equal(t, expectedErr, err)
}

func TestShouldDiscoverAPIBasedOnTags(t *testing.T) {
	tests := []struct {
		name           string
		discoveryTags  []string
		ignoreTags     []string
		apiTags        []string
		expected       bool
		endpoint       string
		lastActiveDate string
	}{
		{
			name:           "Should discover if matching discovery tag exists on API",
			discoveryTags:  []string{"discover"},
			ignoreTags:     []string{},
			apiTags:        []string{"discover"},
			expected:       true,
			endpoint:       "abc.com",
			lastActiveDate: "2021-06-10T21:03:15.706Z",
		},
		{
			name:           "Should not discover if API has a tag to be ignored",
			discoveryTags:  []string{"discover"},
			ignoreTags:     []string{"donotdiscover"},
			apiTags:        []string{"donotdiscover"},
			expected:       false,
			endpoint:       "abc.com",
			lastActiveDate: "2021-06-10T21:03:15.706Z",
		},
		{
			name:           "Should not discover if API does not have any tags that the agent's config has",
			ignoreTags:     []string{"donotdiscover"},
			discoveryTags:  []string{"discover"},
			apiTags:        []string{},
			expected:       false,
			endpoint:       "abc.com",
			lastActiveDate: "2021-06-10T21:03:15.706Z",
		},
		{
			name:           "Should discover if API as well as agent's config have no discovery tags",
			discoveryTags:  []string{},
			ignoreTags:     []string{},
			apiTags:        []string{},
			expected:       true,
			endpoint:       "abc.com",
			lastActiveDate: "2021-06-10T21:03:15.706Z",
		},
		{
			name:           "Should not discover if API has both - a tag to be discovered and a tag to be ignored",
			discoveryTags:  []string{"discover"},
			ignoreTags:     []string{"donotdiscover"},
			apiTags:        []string{"discover", "donotdiscover"},
			expected:       false,
			endpoint:       "abc.com",
			lastActiveDate: "2021-06-10T21:03:15.706Z",
		},
		{
			name:           "Should not discover if the endpoint is empty",
			discoveryTags:  []string{"discover"},
			ignoreTags:     []string{"donotdiscover"},
			apiTags:        []string{"discover"},
			expected:       false,
			endpoint:       "",
			lastActiveDate: "2021-06-10T21:03:15.706Z",
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			ok, _ := shouldDiscoverAPI(tc.endpoint, tc.discoveryTags, tc.ignoreTags, tc.apiTags)
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
	s1 := makeChecksum(&asset, apic.Passthrough)
	s2 := makeChecksum(&asset, common.ClientIDEnforcement)
	assert.NotEmpty(t, s1)
	assert.NotEqual(t, s1, s2)
}

func Test_getAuthPolicy(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		policies []anypoint.Policy
	}{
		{
			name:     "should return the policy as APIKey when the mulesoft policy is client-id-enforcement",
			expected: apic.Apikey,
			policies: []anypoint.Policy{
				{
					Configuration:    map[string]interface{}{},
					PolicyTemplateID: common.ClientIDEnforcement,
				},
			},
		},
		{
			name:     "should return the policy as OAuth when the mulesoft policy is oauth",
			expected: apic.Oauth,
			policies: []anypoint.Policy{
				{
					Configuration:    map[string]interface{}{},
					PolicyTemplateID: common.ExternalOauth,
				},
			},
		},
		{
			name:     "should return the first policy that matches 'client-id-enforcement'",
			expected: apic.Apikey,
			policies: []anypoint.Policy{
				{
					Configuration:    map[string]interface{}{},
					PolicyTemplateID: "fake",
				},
				{
					Configuration:    map[string]interface{}{},
					PolicyTemplateID: common.ClientIDEnforcement,
				},
			},
		},
		{
			name:     "should return a map for the configuration when it is not set.'",
			expected: apic.Apikey,
			policies: []anypoint.Policy{
				{
					PolicyTemplateID: common.ClientIDEnforcement,
				},
			},
		},
		{
			name:     "should return the policy as pass-through when there are no policies in the array",
			expected: apic.Passthrough,
			policies: []anypoint.Policy{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy, conf, _ := getAuthPolicy(tc.policies, catalog)
			assert.Equal(t, policy, tc.expected)
			assert.NotNil(t, conf)
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
			endpoint:        "https://newhost.com/v1",
			content:         []byte(`{"basePath": "/v2","host": "oldhost.com","schemes": ["http"],"swagger": "2.0","info": {"title": "petstore2"},"paths": {}}`),
			expectedContent: []byte(`{"basePath":"/v1","host":"newhost.com","info":{"title":"petstore2","version":""},"schemes":["https"],"securityDefinitions":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e \n\n client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e\n\n","in":"header","name":"authorization","type":"apiKey"}},"swagger":"2.0"}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:            "should update an OAS 2 spec with OAuth security",
			specType:        apic.Oas2,
			endpoint:        "https://newhost.com/v1",
			content:         []byte(`{"basePath":"/v2","host":"oldhost.com","schemes":["http"],"swagger":"2.0","info":{"title":"petstore2"},"paths":{}}`),
			expectedContent: []byte(`{"basePath":"/v1","host":"newhost.com","info":{"title":"petstore2","version":""},"schemes":["https"],"securityDefinitions":{"oauth2":{"description":"This API supports OAuth 2.0 for authenticating all API requests","flow":"accessCode","type":"oauth2"}},"swagger":"2.0"}`),
			authPolicy:      apic.Oauth,
		},
		{
			name:            "should update an OAS 3 spec with OAuth security",
			specType:        apic.Oas3,
			endpoint:        "https://abc.com",
			content:         []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}],"paths":{},"info":{"title":"petstore3"}}`),
			expectedContent: []byte(`{"components":{"securitySchemes":{"oauth2":{"description":"This API supports OAuth 2.0 for authenticating all API requests","flows":{"authorizationCode":{"scopes":{}}},"type":"oauth2"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":{},"servers":[{"url":"https://abc.com"}]}`),
			authPolicy:      apic.Oauth,
		},
		{
			name:            "should update an OAS 3 spec with APIKey security",
			specType:        apic.Oas3,
			endpoint:        "https://abc.com",
			content:         []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}],"paths":{},"info":{"title":"petstore3"}}`),
			expectedContent: []byte(`{"components":{"securitySchemes":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e \n\n client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e\n\n","in":"header","name":"authorization","type":"apiKey"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":{},"servers":[{"url":"https://abc.com"}]}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:            "should update a Raml spec",
			specType:        apic.Raml,
			endpoint:        "https://abc.com",
			content:         []byte("baseUri: https://na1.salesforce.com:4000/services/data/{version}/chatter"),
			expectedContent: []byte("baseUri: https://abc.com\n"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := updateSpec(tc.specType, tc.endpoint, tc.authPolicy, nil, tc.content)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedContent, content)
		})
	}
}

func Test_setOAS2policies(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]interface{}
		content         *openapi2.T
		expectedContent []byte
		authPolicy      string
	}{
		{
			name:          "should apply APIKey security policy with no configuration",
			configuration: nil,
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "oldhost.com",
				BasePath: "/v2",
				Paths:    nil,
			},
			expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","info":{"title":"petstore2","version":""},"schemes":["http"],"securityDefinitions":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e \n\n client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e\n\n","in":"header","name":"authorization","type":"apiKey"}},"swagger":"2.0"}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:          "should apply APIKey security policy with Custom configuration set as Basic Auth",
			configuration: map[string]interface{}{common.CredOrigin: "httpBasicAuthenticationHeader"},
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "oldhost.com",
				BasePath: "/v2",
				Paths:    nil,
			},
			expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","info":{"title":"petstore2","version":""},"schemes":["http"],"securityDefinitions":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e \n\n client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e\n\nhttpBasicAuthenticationHeader","in":"header","name":"authorization","type":"apiKey"}},"swagger":"2.0"}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:          "should apply OAuth security policy with no scope",
			configuration: map[string]interface{}{common.TokenURL: "www.test.com"},
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "oldhost.com",
				BasePath: "/v2",
				Paths:    nil,
			},
			expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","info":{"title":"petstore2","version":""},"schemes":["http"],"securityDefinitions":{"oauth2":{"authorizationUrl":"www.test.com","description":"This API supports OAuth 2.0 for authenticating all API requests","flow":"accessCode","tokenUrl":"www.test.com","type":"oauth2"}},"swagger":"2.0"}`),
			authPolicy:      apic.Oauth,
		},
		{
			name:          "should apply OAuth security policy with scopes",
			configuration: map[string]interface{}{common.TokenURL: "www.test.com", common.Scopes: "read,write"},
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "oldhost.com",
				BasePath: "/v2",
				Paths:    nil,
			},
			expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","info":{"title":"petstore2","version":""},"schemes":["http"],"securityDefinitions":{"oauth2":{"authorizationUrl":"www.test.com","description":"This API supports OAuth 2.0 for authenticating all API requests","flow":"accessCode","scopes":{"scopes":"read,write"},"tokenUrl":"www.test.com","type":"oauth2"}},"swagger":"2.0"}`),
			authPolicy:      apic.Oauth,
		},
		// {
		// 	name:          "should return error when authPolicy type is not supported ",
		// 	configuration: nil,
		// 	content: &openapi2.T{
		// 		Swagger: "2.0",
		// 		Info: openapi3.Info{
		// 			Title: "petstore2",
		// 		},
		// 		Schemes:  []string{"http"},
		// 		Host:     "oldhost.com",
		// 		BasePath: "/v2",
		// 		Paths:    nil,
		// 	},
		// 	expectedContent: []byte(`{"basePath":"/v2","host":"oldhost.com","schemes":["http"],"swagger":"2.0"}`),
		// 	authPolicy:      "JWTToken",
		// },
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := setOAS2policies(tc.content, tc.authPolicy, tc.configuration)
			if tc.authPolicy != apic.Oauth && tc.authPolicy != apic.Apikey {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedContent, content)

			}
		})
	}
}

func Test_setOAS3policies(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]interface{}
		content         *openapi3.T
		expectedContent []byte
		authPolicy      string
	}{
		{
			name:          "should apply APIKey security policy with no configuration",
			configuration: nil,
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: []byte(`{"components":{"securitySchemes":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e \n\n client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e\n\n","in":"header","name":"authorization","type":"apiKey"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":null,"servers":[{"url":"http://google.com"}]}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:          "should apply APIKey security policy with Custom configuration set as Basic Auth",
			configuration: map[string]interface{}{common.CredOrigin: "httpBasicAuthenticationHeader"},
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: []byte(`{"components":{"securitySchemes":{"client-id-enforcement":{"description":"Provided as: client_id:\u003cINSERT_VALID_CLIENTID_HERE\u003e \n\n client_secret:\u003cINSERT_VALID_SECRET_HERE\u003e\n\nhttpBasicAuthenticationHeader","in":"header","name":"authorization","type":"apiKey"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":null,"servers":[{"url":"http://google.com"}]}`),
			authPolicy:      apic.Apikey,
		},
		{
			name:          "should apply OAuth security policy with no scope",
			configuration: map[string]interface{}{common.TokenURL: "www.test.com"},
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: []byte(`{"components":{"securitySchemes":{"oauth2":{"description":"This API supports OAuth 2.0 for authenticating all API requests","flows":{"authorizationCode":{"authorizationUrl":"www.test.com","scopes":{},"tokenUrl":"www.test.com"}},"type":"oauth2"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":null,"servers":[{"url":"http://google.com"}]}`),
			authPolicy:      apic.Oauth,
		},
		{
			name:          "should apply OAuth security policy with scopes",
			configuration: map[string]interface{}{common.TokenURL: "www.test.com", common.Scopes: "read,write"},
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: []byte(`{"components":{"securitySchemes":{"oauth2":{"description":"This API supports OAuth 2.0 for authenticating all API requests","flows":{"authorizationCode":{"authorizationUrl":"www.test.com","scopes":{"scopes":"read,write"},"tokenUrl":"www.test.com"}},"type":"oauth2"}}},"info":{"title":"petstore3","version":""},"openapi":"3.0.1","paths":null,"servers":[{"url":"http://google.com"}]}`),
			authPolicy:      apic.Oauth,
		},
		// {
		// 	name:            "should return error when authPolicy type is not supported",
		// 	configuration:   nil,
		// 	content:         []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}]}`),
		// 	expectedContent: []byte(`{"openapi":"3.0.1","servers":[{"url":"google.com"}]}`),
		// 	authPolicy:      "JWTToken",
		// },
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content, err := setOAS3policies(tc.content, tc.authPolicy, tc.configuration)
			if tc.authPolicy != apic.Oauth && tc.authPolicy != apic.Apikey {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedContent, content)
			}
		})
	}
}

func getSLATierInfo() (*anypoint.Tiers, *serviceHandler, *mocks.MockCentralClient) {
	stage := "Sandbox"
	mc := &anypoint.MockAnypointClient{}
	tiers := anypoint.Tiers{
		Total: 2,
		Tiers: []anypoint.SLATier{{
			ID:   123,
			Name: "Gold",
			Limits: []anypoint.Limits{{
				TimePeriodInMilliseconds: 1000,
				MaximumRequests:          10,
			}},
		}, {
			ID:   456,
			Name: "Silver",
			Limits: []anypoint.Limits{{
				TimePeriodInMilliseconds: 1000,
				MaximumRequests:          1,
			}},
		}},
	}

	msc := &subscription.MockMuleSubscriptionClient{}
	sm := subscription.NewManager(logrus.StandardLogger(), msc)

	sh := &serviceHandler{
		muleEnv:             stage,
		discoveryTags:       []string{},
		discoveryIgnoreTags: []string{},
		client:              mc,
		schemas:             sm,
		cache:               cache.New(),
	}

	return &tiers, sh, &mocks.MockCentralClient{}
}

func TestCreateSubscriptionSchemaForSLATier(t *testing.T) {
	tiers, sh, mcc := getSLATierInfo()

	_, err := sh.createSLATierSchema("1", tiers, mcc)
	assert.Nil(t, err)
}

func TestSLATierSchemaSubscriptionCreateFailure(t *testing.T) {
	tiers, sh, mcc := getSLATierInfo()

	mcc.RegisterSubscriptionSchemaMock = func(_ apic.SubscriptionSchema, _ bool) error {
		return fmt.Errorf("cannot register subscription schema")
	}
	_, err := sh.createSLATierSchema("1", tiers, mcc)

	assert.Error(t, err)
}
