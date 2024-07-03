package discovery

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Axway/agents-mulesoft/pkg/common"
	"gopkg.in/yaml.v2"

	"github.com/Axway/agent-sdk/pkg/cache"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"

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
		expectedResourceType string
	}
	cases := []testCase{
		{
			content: "#%RAML 1.0\ntitle: API with Examples\ndescription: Grand Theft Auto:Vice City\nversion: v3\nprotocols: [HTTP,HTTPS]\nbaseUri: https://na1.salesforce.com:4000/services/data/{version}/chatter",
			policies: []anypoint.Policy{
				{
					PolicyTemplateID: common.ClientIDEnforcement,
				},
			},
			exchangeAsset:        &exchangeAsset,
			expectedResourceType: apic.Raml,
		},
		{
			content: `{"openapi":"3.0.1","servers":[{"url":"https://abc.com"}], "paths":{}, "info":{"title":"petstore3"}}`,
			policies: []anypoint.Policy{
				{
					PolicyTemplateID: common.ClientIDEnforcement,
				},
			},
			exchangeAsset:        &exchangeAsset,
			expectedResourceType: apic.Oas3,
		},
	}
	for _, c := range cases {
		mc := &anypoint.MockAnypointClient{}
		mc.On("GetPolicies").Return(c.policies, nil)
		mc.On("GetExchangeAsset").Return(c.exchangeAsset, nil)
		mc.On("GetExchangeFileContent").Return([]byte(c.content), true, nil)
		mc.On("GetExchangeAssetIcon").Return("", "", nil)
		mc.On("GetAPI").Return(&asset.APIs[0], nil)

		sh := &serviceHandler{
			muleEnv:             "Sandbox",
			discoveryTags:       []string{"tag1"},
			discoveryIgnoreTags: []string{"nah"},
			client:              mc,
			cache:               cache.New(),
		}
		list := sh.ToServiceDetails(&asset)
		api := asset.APIs[0]
		assert.Equal(t, 1, len(list))
		item := list[0]

		assert.Equal(t, asset.APIs[0].AssetID, item.APIName)
		assert.Equal(t, "", item.AuthPolicy)
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
	}
	details := sh.ToServiceDetails(&asset)
	assert.Equal(t, 0, len(details))
	assert.Equal(t, 0, len(mc.Calls))
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
		name                 string
		files                []anypoint.ExchangeFile
		expected             *anypoint.ExchangeFile
		discoverOriginalRaml bool
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
		{
			name: "Should sort files and return first non-empty mainFile",
			files: []anypoint.ExchangeFile{
				{
					Classifier: "wsdl",
				},
				{
					Classifier: "oas",
				},
				{
					Classifier: "raml",
					MainFile:   "1",
				},
			},
			expected: &anypoint.ExchangeFile{
				Classifier: "raml",
				MainFile:   "1",
			},
			discoverOriginalRaml: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sd := getExchangeAssetSpecFile(tc.files, tc.discoverOriginalRaml)
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

func Test_getAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		policies []anypoint.Policy
		err      error
	}{
		{
			name:     "OAuth2Policy",
			expected: []string{apic.Oauth},
			policies: []anypoint.Policy{
				{
					PolicyTemplateID: common.OAuth2MuleOauthProviderPolicy,
				},
			},
		},
		{
			name:     "BasicAuthSimplePolicy",
			expected: []string{apic.Basic},
			policies: []anypoint.Policy{
				{
					PolicyTemplateID: common.BasicAuthSimplePolicy,
				},
			},
		},
		{
			name:     "ClientIDEnforcement",
			expected: []string{apic.Basic},
			policies: []anypoint.Policy{
				{
					Configuration: map[string]interface{}{
						common.CredOrigin: "sth",
					},
					PolicyTemplateID: common.ClientIDEnforcementPolicy,
				},
			},
		},
		{
			name:     "ClientIDEnforcementCustomExpression",
			expected: []string{},
			policies: []anypoint.Policy{
				{
					Configuration: map[string]interface{}{
						common.CredOrigin: "customExpression",
					},
					PolicyTemplateID: common.ClientIDEnforcementPolicy,
				},
			},
			err: fmt.Errorf("incompatible Mulesoft Policies provided"),
		},
		{
			name:     "BasicAuth_ClientIDEnforcement",
			expected: []string{apic.Oauth, apic.Basic},
			policies: []anypoint.Policy{
				{
					PolicyTemplateID: common.OAuth2MuleOauthProviderPolicy,
				},
				{
					PolicyTemplateID: common.BasicAuthSimplePolicy,
				},
			},
		},
		{
			name:     "Passthrough",
			expected: []string{},
			policies: []anypoint.Policy{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			apicAuths, conf, err := getApicAuthsAndConfig(tc.policies)
			if tc.err != nil {
				assert.Equal(t, err, tc.err)
				return
			}
			assert.Equal(t, err, nil)
			assert.Equal(t, tc.expected, apicAuths)
			assert.NotNil(t, conf)
		})
	}
}

func TestSetPolicies(t *testing.T) {
	urlExample := "https://www.test.com"
	tests := []struct {
		name            string
		configuration   map[string]interface{}
		content         interface{}
		expectedContent map[string]interface{}
	}{
		{
			name: "OAS3_HttpBasic",
			configuration: map[string]interface{}{
				apic.Basic: "",
			},
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Paths:   &openapi3.Paths{},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: map[string]interface{}{
				"components": map[string]interface{}{
					"securitySchemes": map[string]interface{}{
						common.BasicAuthName: map[string]interface{}{
							"description": common.BasicAuthDesc,
							"scheme":      common.BasicAuthScheme,
							"type":        common.BasicAuthOASType,
						},
					},
				},
				"info": map[string]interface{}{
					"title":   "petstore3",
					"version": "",
				},
				"openapi": "3.0.1",
				"paths":   map[string]interface{}{},
				"servers": []interface{}{
					map[string]interface{}{
						"url": "http://google.com",
					},
				},
				"security": []map[string]interface{}{
					map[string]interface{}{
						common.BasicAuthName: []interface{}{""},
					},
				},
			},
		},

		{
			name: "OAS3_Oauth2",
			configuration: map[string]interface{}{
				apic.Oauth: map[string]interface{}{
					common.TokenURL: "www.test.com",
				},
			},
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Paths:   &openapi3.Paths{},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: map[string]interface{}{
				"components": map[string]interface{}{
					"securitySchemes": map[string]interface{}{
						common.Oauth2Name: map[string]interface{}{
							"description": common.Oauth2Desc,
							"type":        common.Oauth2OASType,
							"flows": map[string]interface{}{
								"clientCredentials": map[string]interface{}{
									"scopes":   map[string]interface{}{},
									"tokenUrl": "www.test.com",
								},
							},
						},
					},
				},
				"info": map[string]interface{}{
					"title":   "petstore3",
					"version": "",
				},
				"openapi": "3.0.1",
				"paths":   map[string]interface{}{},
				"security": []interface{}{
					map[string]interface{}{
						common.Oauth2Name: []interface{}{},
					},
				},
				"servers": []interface{}{
					map[string]interface{}{
						"url": "http://google.com",
					},
				},
			},
		},

		{
			name: "OAS3_Oauth2_Scopes",
			configuration: map[string]interface{}{
				apic.Oauth: map[string]interface{}{
					common.TokenURL: "www.test.com", common.Scopes: "read write",
				},
			},
			content: &openapi3.T{
				OpenAPI: "3.0.1",
				Info: &openapi3.Info{
					Title: "petstore3",
				},
				Paths:   &openapi3.Paths{},
				Servers: openapi3.Servers{{URL: "http://google.com"}},
			},
			expectedContent: map[string]interface{}{
				"components": map[string]interface{}{
					"securitySchemes": map[string]interface{}{
						common.Oauth2Name: map[string]interface{}{
							"description": common.Oauth2Desc,
							"type":        common.Oauth2OASType,
							"flows": map[string]interface{}{
								"clientCredentials": map[string]interface{}{
									"scopes": map[string]interface{}{
										"read":  "",
										"write": "",
									},
									"tokenUrl": "www.test.com",
								},
							},
						},
					},
				},
				"info": map[string]interface{}{
					"title":   "petstore3",
					"version": "",
				},
				"openapi": "3.0.1",
				"paths":   map[string]interface{}{},
				"security": []interface{}{
					map[string]interface{}{
						common.Oauth2Name: []interface{}{
							"read",
							"write",
						},
					},
				},
				"servers": []interface{}{
					map[string]interface{}{
						"url": "http://google.com",
					},
				},
			},
		},

		{
			name: "OAS2_HttpBasic",
			configuration: map[string]interface{}{
				apic.Basic: "",
			},
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "www.test.com",
				BasePath: "/v2",
			},
			expectedContent: map[string]interface{}{
				"basePath": "/v2",
				"host":     "www.test.com",
				"info": map[string]interface{}{
					"title":   "petstore2",
					"version": "",
				},
				"schemes": []interface{}{
					"http",
				},
				"security": []map[string]interface{}{
					map[string]interface{}{
						common.BasicAuthName: []interface{}{},
					},
				},
				"securityDefinitions": map[string]interface{}{
					common.BasicAuthName: map[string]interface{}{
						"description": common.BasicAuthDesc,
						"type":        common.BasicAuthScheme,
					},
				},
				"swagger": "2.0",
			},
		},

		{
			name: "OAS2_Oauth2",
			configuration: map[string]interface{}{
				apic.Oauth: map[string]interface{}{
					common.TokenURL: "www.test.com",
				},
			},
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "www.test.com",
				BasePath: "/v2",
			},
			expectedContent: map[string]interface{}{
				"basePath": "/v2",
				"host":     "www.test.com",
				"info": map[string]interface{}{
					"title":   "petstore2",
					"version": "",
				},
				"schemes": []interface{}{
					"http",
				},
				"security": []interface{}{
					map[string]interface{}{
						common.Oauth2Name: []interface{}{},
					},
				},
				"securityDefinitions": map[string]interface{}{
					common.Oauth2Name: map[string]interface{}{
						"description": common.Oauth2Desc,
						"flow":        common.ClientCredentials,
						"tokenUrl":    "www.test.com",
						"type":        common.Oauth2OASType,
					},
				},
				"swagger": "2.0",
			},
		},

		{
			name: "OAS2_Oauth2_Scopes",
			configuration: map[string]interface{}{
				apic.Oauth: map[string]interface{}{
					common.TokenURL: "www.test.com", common.Scopes: "read write",
				},
			},
			content: &openapi2.T{
				Swagger: "2.0",
				Info: openapi3.Info{
					Title: "petstore2",
				},
				Schemes:  []string{"http"},
				Host:     "www.test.com",
				BasePath: "/v2",
			},
			expectedContent: map[string]interface{}{
				"basePath": "/v2",
				"host":     "www.test.com",
				"info": map[string]interface{}{
					"title":   "petstore2",
					"version": "",
				},
				"schemes": []interface{}{
					"http",
				},
				"security": []interface{}{
					map[string]interface{}{
						common.Oauth2Name: []interface{}{
							"read",
							"write",
						},
					},
				},
				"securityDefinitions": map[string]interface{}{
					common.Oauth2Name: map[string]interface{}{
						"description": common.Oauth2Desc,
						"flow":        common.ClientCredentials,
						"scopes": map[string]interface{}{
							"read":  "",
							"write": "",
						},
						"tokenUrl": "www.test.com",
						"type":     common.Oauth2OASType,
					},
				},
				"swagger": "2.0",
			},
		},

		{
			name: "RAML_Basic",
			configuration: map[string]interface{}{
				apic.Basic: "",
			},
			content: []byte("#%RAML 1.0\ntitle: ok"),
			expectedContent: map[string]interface{}{
				"baseUri":   urlExample,
				"securedBy": []interface{}{common.BasicAuthName},
				"securitySchemes": map[string]interface{}{
					common.BasicAuthName: map[string]interface{}{
						"description": common.BasicAuthDesc,
						"type":        common.BasicAuthRAMLType,
					},
				},
				"title": "ok",
			},
		},

		{
			name: "RAML_OAuth2",
			configuration: map[string]interface{}{
				apic.Oauth: map[string]interface{}{
					common.TokenURL: urlExample,
				},
			},
			content: []byte("#%RAML 1.0\ntitle: ok"),
			expectedContent: map[string]interface{}{
				"baseUri": urlExample,
				"securedBy": []interface{}{
					common.Oauth2Name,
				},
				"securitySchemes": map[string]interface{}{
					common.Oauth2Name: map[string]interface{}{
						"description": common.Oauth2Desc,
						"type":        common.Oauth2RAMLType,
						"settings": map[string]interface{}{
							"accessTokenUri": urlExample,
						},
						"describedBy": map[string]interface{}{
							"headers": map[string]interface{}{
								"Authorization": map[string]interface{}{
									"description": common.Oauth2Desc,
									"type":        "string",
								},
							},
						},
					},
				},
				"title": "ok",
			},
		},

		{
			name: "RAML_OAuth2_Scopes",
			configuration: map[string]interface{}{
				apic.Oauth: map[string]interface{}{
					common.TokenURL: urlExample, common.Scopes: "read write",
				},
			},
			content: []byte("#%RAML 1.0\ntitle: ok"),
			expectedContent: map[string]interface{}{
				"baseUri": urlExample,
				"securedBy": []interface{}{
					map[string]interface{}{
						common.Oauth2Name: map[string]interface{}{
							"scopes": []interface{}{
								"read",
								"write",
							},
						},
					},
				},
				"securitySchemes": map[string]interface{}{
					common.Oauth2Name: map[string]interface{}{
						"description": common.Oauth2Desc,
						"type":        common.Oauth2RAMLType,
						"settings": map[string]interface{}{
							"accessTokenUri": urlExample,
						},
						"describedBy": map[string]interface{}{
							"headers": map[string]interface{}{
								"Authorization": map[string]interface{}{
									"description": common.Oauth2Desc,
									"type":        "string",
								},
							},
						},
					},
				},
				"title": "ok",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := []byte{}
			var err error
			expected := []byte{}
			switch content := tc.content.(type) {
			case *openapi3.T:
				actual, err = setOAS3policies(content, tc.configuration)
				expected, _ = json.Marshal(tc.expectedContent)
			case *openapi2.T:
				actual, err = setOAS2policies(content, tc.configuration)
				expected, _ = json.Marshal(tc.expectedContent)
			case []byte:
				actual, err = setRamlHostAndAuth(content, urlExample, tc.configuration)
				expected, _ = yaml.Marshal(tc.expectedContent)
				expected = append([]byte("#%RAML 1.0\n"), expected...)
			}
			assert.Nil(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}
