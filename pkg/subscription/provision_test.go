package subscription

import (
	"fmt"
	"testing"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/mock"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAccessRequestDeprovision(t *testing.T) {
	tests := []struct {
		name       string
		apiID      string
		contractID string
		status     prov.Status
	}{
		{
			name:       "should deprovision an access request",
			apiID:      "1234",
			contractID: "5432",
			status:     prov.Success,
		},
		{
			name:       "should return an error when the apiID is not found",
			contractID: "5432",
			status:     prov.Error,
		},
		{
			name:   "should return an error when the contractID is not found",
			apiID:  "1234",
			status: prov.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &MockMuleSubscriptionClient{}
			prv := NewProvisioner(client, logrus.StandardLogger())
			req := mock.MockAccessRequest{
				AppName: "app1",
				Details: map[string]string{
					common.ContractID: tc.contractID,
				},
				InstanceDetails: map[string]interface{}{
					common.AttrAPIID: tc.apiID,
				},
			}
			status := prv.AccessRequestDeprovision(req)
			assert.Equal(t, tc.status.String(), status.GetStatus().String())
		})
	}
}

func TestAccessRequestProvision(t *testing.T) {
	tests := []struct {
		name    string
		status  prov.Status
		apiID   string
		stage   string
		appID   string
		slaTier string
		err     error
	}{
		{
			name:    "should provision an access request",
			status:  prov.Success,
			apiID:   "111",
			stage:   "v1",
			appID:   "65432",
			slaTier: "1143480-free",
		},
		{
			name:    "should return an error when provisioning",
			status:  prov.Error,
			apiID:   "111",
			stage:   "v1",
			appID:   "65432",
			slaTier: "1143480-free",
			err:     fmt.Errorf("failed to provision"),
		},
		{
			name:    "should return an error when apiID is not found",
			status:  prov.Error,
			apiID:   "",
			stage:   "v1",
			appID:   "65432",
			slaTier: "1143480-free",
		},
		{
			name:    "should return an error when the stage is not found",
			status:  prov.Error,
			apiID:   "111",
			stage:   "",
			appID:   "65432",
			slaTier: "1143480-free",
		},
		{
			name:    "should return an error when the appID is not found",
			status:  prov.Error,
			apiID:   "111",
			stage:   "v1",
			appID:   "",
			slaTier: "1143480-free",
		},
		{
			name:    "should provision with no error without an sla tier",
			status:  prov.Success,
			apiID:   "111",
			stage:   "v1",
			appID:   "65432",
			slaTier: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			contract := &anypoint.Contract{
				Id: 98765,
			}

			client := &MockMuleSubscriptionClient{
				app:      nil,
				err:      tc.err,
				contract: contract,
			}

			prv := NewProvisioner(client, logrus.StandardLogger())

			req := &mock.MockAccessRequest{
				AppDetails: map[string]string{
					common.AppID: tc.appID,
				},
				AppName: "",

				InstanceDetails: map[string]interface{}{
					common.AttrAPIID:          tc.apiID,
					defs.AttrExternalAPIStage: tc.stage,
				},
				AccessRequestData: map[string]interface{}{
					common.SlaTier: tc.slaTier,
				},
			}

			status, _ := prv.AccessRequestProvision(req)
			assert.Equal(t, tc.status.String(), status.GetStatus().String())
			if tc.status == prov.Success {
				assert.Contains(t, status.GetProperties(), common.ContractID)
			} else {
				assert.Empty(t, status.GetProperties(), common.ContractID)
			}
		})
	}
}

func TestApplicationRequestDeprovision(t *testing.T) {
	tests := []struct {
		name   string
		appID  string
		status prov.Status
		err    error
	}{
		{
			name:   "should deprovision an application",
			appID:  "65432",
			status: prov.Success,
		},
		{
			name:   "should fail to deprovision an application",
			appID:  "65432",
			status: prov.Error,
			err:    fmt.Errorf("failed to deprovision"),
		},
		{
			name:   "should return an error when the appID is not found",
			appID:  "",
			status: prov.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &MockMuleSubscriptionClient{
				err: tc.err,
			}
			prv := NewProvisioner(client, logrus.StandardLogger())
			req := mock.MockApplicationRequest{
				AppName: "app1",
				Details: map[string]string{
					common.AppID: tc.appID,
				},
			}
			status := prv.ApplicationRequestDeprovision(req)
			assert.Equal(t, tc.status.String(), status.GetStatus().String())
		})
	}
}

func TestApplicationRequestProvision(t *testing.T) {
	tests := []struct {
		name    string
		status  prov.Status
		err     error
		appName string
	}{
		{
			name:    "should provision an application",
			appName: "app1",
			status:  prov.Success,
		},
		{
			name:   "should return an error when the appName is not found",
			status: prov.Error,
		},
		{
			name:    "should return an error when the appName is not found",
			err:     fmt.Errorf("fail to deprovision"),
			appName: "app1",
			status:  prov.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := &anypoint.Application{
				APIEndpoints: false,
				ID:           65432,
				Name:         tc.appName,
			}
			client := &MockMuleSubscriptionClient{
				err: tc.err,
				app: app,
			}
			prv := NewProvisioner(client, logrus.StandardLogger())
			req := mock.MockApplicationRequest{
				AppName: tc.appName,
			}
			status := prv.ApplicationRequestProvision(req)
			assert.Equal(t, tc.status.String(), status.GetStatus().String())
			if tc.status == prov.Success {
				assert.Contains(t, status.GetProperties(), common.AppID)
			} else {
				assert.Empty(t, status.GetProperties(), common.AppID)
			}
		})
	}
}

func TestCredentialDeprovision(t *testing.T) {
	client := &MockMuleSubscriptionClient{}
	prv := NewProvisioner(client, logrus.StandardLogger())
	req := mock.MockCredentialRequest{}
	status := prv.CredentialDeprovision(req)
	assert.Equal(t, prov.Success.String(), status.GetStatus().String())
}

func TestCredentialProvision(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		appID   string
		err     error
		status  prov.Status
	}{
		{
			name:    "should provision credentials",
			appName: "app1",
			appID:   "65432",
			status:  prov.Success,
		},
		{
			name:    "should fail to provision credentials",
			appName: "app1",
			appID:   "65432",
			err:     fmt.Errorf("failed to provision"),
			status:  prov.Error,
		},
		{
			name:    "should return an error when appID is not found",
			appName: "app1",
			appID:   "",
			status:  prov.Error,
		},
		{
			name:   "should return an error when appName is not found",
			appID:  "65432",
			status: prov.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := &anypoint.Application{
				ClientID:     "12345",
				ClientSecret: "lajksdf",
			}
			client := &MockMuleSubscriptionClient{
				err: tc.err,
				app: app,
			}
			prv := NewProvisioner(client, logrus.StandardLogger())
			req := mock.MockCredentialRequest{
				AppName: tc.appName,
				AppDetails: map[string]string{
					common.AppID: tc.appID,
				},
			}
			status, cr := prv.CredentialProvision(req)
			assert.Equal(t, tc.status.String(), status.GetStatus().String())
			if tc.status.String() == prov.Success.String() {
				assert.NotNil(t, cr)
				assert.Contains(t, cr.GetData(), prov.OauthClientSecret)
				assert.Contains(t, cr.GetData(), prov.OauthClientID)
			} else {
				assert.Nil(t, cr)
			}
		})
	}
}

func TestCredentialUpdate(t *testing.T) {
	tests := []struct {
		name      string
		appName   string
		appID     string
		getAppErr error
		rotateErr error
		status    prov.Status
		action    prov.CredentialAction
	}{
		{
			name:    "should update credentials",
			appName: "app1",
			appID:   "65432",
			status:  prov.Success,
			action:  prov.Rotate,
		},
		{
			name:    "should fail to update credentials when the action is not rotate",
			appName: "app1",
			appID:   "65432",
			status:  prov.Error,
			action:  prov.Suspend,
		},
		{
			name:      "should fail to update credentials when making the api call",
			appName:   "app1",
			appID:     "65432",
			rotateErr: fmt.Errorf("error"),
			status:    prov.Error,
			action:    prov.Rotate,
		},
		{
			name:      "should return an error when app is not found",
			appName:   "app1",
			appID:     "65432",
			status:    prov.Error,
			action:    prov.Rotate,
			getAppErr: fmt.Errorf("failed to get app"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := &anypoint.Application{
				ClientID:     "12345",
				ClientSecret: "lajksdf",
			}
			newApp := &anypoint.Application{
				ClientID:     "12345",
				ClientSecret: "uihgobfjd",
			}
			client := &MockMuleSubscriptionClient{
				err:       tc.getAppErr,
				rotateErr: tc.rotateErr,
				app:       app,
				newApp:    newApp,
			}
			prv := NewProvisioner(client, logrus.StandardLogger())
			req := mock.MockCredentialRequest{
				AppName: tc.appName,
				AppDetails: map[string]string{
					common.AppID: tc.appID,
				},
				Action: tc.action,
			}

			status, cr := prv.CredentialUpdate(req)
			assert.Equal(t, tc.status.String(), status.GetStatus().String())
			if tc.status.String() == prov.Success.String() {
				assert.NotNil(t, cr)
				assert.Contains(t, cr.GetData(), prov.OauthClientSecret)
				assert.Contains(t, cr.GetData(), prov.OauthClientID)
				assert.NotEqual(t, app.ClientSecret, cr.GetData()[prov.OauthClientSecret])
			} else {
				assert.Nil(t, cr)
			}
		})
	}
}
