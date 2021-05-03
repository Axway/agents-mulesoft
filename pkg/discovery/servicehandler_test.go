package discovery

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

func TestServiceHandler(t *testing.T) {
	t.Skip()
	mc := &mockAnypointClient{}
	sh := &serviceHandler{
		assetCache:          cache.New(),
		freshCache:          cache.New(),
		stage:               "Sandbox",
		discoveryTags:       []string{"tag1"},
		discoveryIgnoreTags: []string{"nah"},
		client:              mc,
	}
	details := sh.ToServiceDetails(&asset)
	logrus.Info(details)
}

func TestShouldDiscoverAPIBasedOnTags(t *testing.T) {
	tests := []struct {
		name          string
		discoveryTags []string
		ignoreTags    []string
		apiTags       []string
		expected      bool
	}{
		{
			name:          "Should discover if matching discovery tag exists on API",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{},
			apiTags:       []string{"discover"},
			expected:      true,
		},
		{
			name:          "Should not discover if API has a tag to be ignored",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{"donotdiscover"},
			apiTags:       []string{"donotdiscover"},
			expected:      false,
		},
		{
			name:          "Should not discover if API does not have any tags that the agent's config has",
			ignoreTags:    []string{"donotdiscover"},
			discoveryTags: []string{"discover"},
			apiTags:       []string{},
			expected:      false,
		},
		{
			name:          "Should discover if API as well as agent's config have no discovery tags",
			discoveryTags: []string{},
			ignoreTags:    []string{},
			apiTags:       []string{},
			expected:      true,
		},
		{
			name:          "Should not discover if API has both - a tag to be discovered and a tag to be ignored",
			discoveryTags: []string{"discover"},
			ignoreTags:    []string{"donotdiscover"},
			apiTags:       []string{"discover", "donotdiscover"},
			expected:      false,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			ok := shouldDiscoverAPI(tc.discoveryTags, tc.ignoreTags, tc.apiTags)
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

func TestSetOAS2Endpoint(t *testing.T) {
	tests := []struct {
		name        string
		endPointURL string
		specContent []byte
		result      []byte
		err         error
	}{
		{
			name:        "Should return error if Endpoint URL is not valid",
			endPointURL: "postgres://user:abc{def=ghi@sdf.com:5432",
			specContent: []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
			result:      []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
			err: &url.Error{
				Op:  "parse",
				URL: "postgres://user:abc{def=ghi@sdf.com:5432",
				Err: fmt.Errorf("net/url: invalid userinfo"),
			},
		},
		{
			name:        "Should return error if the spec content is not a valid JSON",
			endPointURL: "http://google.com",
			specContent: []byte("google.com"),
			result:      []byte("google.com"),
			err:         fmt.Errorf("invalid character 'g' looking for beginning of value"),
		},
		{
			name:        "Should return spec that has OAS2 endpoint set",
			endPointURL: "http://google.com",
			specContent: []byte("{\"basePath\":\"google.com\",\"host\":\"\",\"schemes\":[\"\"],\"swagger\":\"2.0\"}"),
			result:      []byte("{\"basePath\":\"\",\"host\":\"google.com\",\"schemes\":[\"http\"],\"swagger\":\"2.0\"}"),
			err:         nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := setOAS2Endpoint(tc.endPointURL, tc.specContent)

			if err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}

			assert.Equal(t, tc.result, spec)
		})
	}
}

func TestSetOAS3Endpoint(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		specContent []byte
		result      []byte
		err         error
	}{
		{
			name:        "Should return error if the spec content is not a valid JSON",
			url:         "google.com",
			specContent: []byte("google.com"),
			result:      []byte("google.com"),
			err:         fmt.Errorf("invalid character 'g' looking for beginning of value"),
		},
		{
			name:        "Should return spec that has OAS3 endpoint set",
			url:         "google.com",
			specContent: []byte("{\"openapi\": \"3.0.1\"}"),
			result:      []byte("{\"openapi\":\"3.0.1\",\"servers\":[{\"url\":\"google.com\"}]}"),
			err:         nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := setOAS3Endpoint(tc.url, tc.specContent)
			if err != nil {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
			assert.Equal(t, tc.result, spec)
		})
	}
}
