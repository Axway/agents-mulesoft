package subscription

import (
	"fmt"
	"strconv"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agents-mulesoft/pkg/common"
	"github.com/sirupsen/logrus"
)

type provisioner struct {
	client MuleSubscriptionClient
	log    logrus.FieldLogger
}

// NewProvisioner creates a type to implement the SDK Provisioning methods for handling subscriptions
func NewProvisioner(client MuleSubscriptionClient, log logrus.FieldLogger) prov.Provisioning {
	return &provisioner{
		client: client,
		log:    log.WithField("component", "mp-provisioner"),
	}
}

// AccessRequestDeprovision deletes a contract
func (p provisioner) AccessRequestDeprovision(req prov.AccessRequest) prov.RequestStatus {
	p.log.Info("deprovisioning access request")
	rs := prov.NewRequestStatusBuilder()
	instDetails := req.GetInstanceDetails()

	apiID := util.ToString(instDetails[common.AttrAPIID])
	if apiID == "" {
		return p.failed(rs, notFound(common.AttrAPIID))
	}

	contractID := req.GetAccessRequestDetailsValue(common.ContractID)
	if contractID == "" {
		return p.failed(rs, notFound(common.ContractID))
	}

	// skip error handling since access request may already be deleted if the managed app was deleted first.
	p.client.DeleteContract(apiID, contractID)

	p.log.
		WithField("api", apiID).
		WithField("app", req.GetApplicationName()).
		Info("removed access")
	return rs.Success()
}

// AccessRequestProvision adds an API to an app
func (p provisioner) AccessRequestProvision(req prov.AccessRequest) (prov.RequestStatus, prov.AccessData) {
	p.log.Info("provisioning access request")
	rs := prov.NewRequestStatusBuilder()
	instDetails := req.GetInstanceDetails()

	apiID := util.ToString(instDetails[common.AttrAPIID])
	if apiID == "" {
		return p.failed(rs, notFound(common.AttrAPIID)), nil
	}

	stage := util.ToString(instDetails[defs.AttrExternalAPIStage])
	if stage == "" {
		return p.failed(rs, notFound(defs.AttrExternalAPIStage)), nil
	}

	appID := req.GetApplicationDetailsValue(common.AppID)
	appID64, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return p.failed(rs, fmt.Errorf("failed to convert appID to int64. %s", err)), nil
	}

	accessData := req.GetAccessRequestData()
	tier := util.ToString(accessData[common.SlaTier])

	contract, err := p.client.CreateContract(apiID, tier, appID64)
	if err != nil {
		return p.failed(rs, fmt.Errorf("failed to create contract: %s", err)), nil
	}

	rs.AddProperty(common.ContractID, fmt.Sprintf("%d", contract.Id))

	p.log.
		WithField("api", apiID).
		WithField("app", req.GetApplicationName()).
		Info("granted access")
	return rs.Success(), nil
}

// ApplicationRequestDeprovision deletes an app
func (p provisioner) ApplicationRequestDeprovision(req prov.ApplicationRequest) prov.RequestStatus {
	p.log.Info("deprovisioning application")
	rs := prov.NewRequestStatusBuilder()

	appID := req.GetApplicationDetailsValue(common.AppID)
	appID64, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return p.failed(rs, fmt.Errorf("failed to convert appID to int64. %s", err))
	}

	err = p.client.DeleteApp(appID64)
	if err != nil {
		return p.failed(rs, fmt.Errorf("failed to delete app: %s", err))
	}

	p.log.
		WithField("appName", req.GetManagedApplicationName()).
		Info("removed application")
	return rs.Success()
}

// ApplicationRequestProvision creates an app
func (p provisioner) ApplicationRequestProvision(req prov.ApplicationRequest) prov.RequestStatus {
	p.log.Info("provisioning application")
	rs := prov.NewRequestStatusBuilder()

	appName := req.GetManagedApplicationName()
	if appName == "" {
		return p.failed(rs, notFound("managed application name"))
	}

	app, err := p.client.CreateApp(appName, "0", "Created by Amplify Mulesoft Agent")
	if err != nil {
		return p.failed(rs, fmt.Errorf("failed to create app: %s", err))
	}

	rs.AddProperty(common.AppID, fmt.Sprintf("%d", app.ID))

	p.log.
		WithField("appName", req.GetManagedApplicationName()).
		Info("created application")

	return rs.Success()
}

// CredentialDeprovision returns success since credentials are removed with the app
func (p provisioner) CredentialDeprovision(_ prov.CredentialRequest) prov.RequestStatus {
	msg := "credentials will be removed when the subscription is deleted"
	p.log.Info(msg)
	return prov.NewRequestStatusBuilder().
		SetMessage("credentials will be removed when the application is deleted").
		Success()
}

// CredentialProvision retrieves the credentials from an app
func (p provisioner) CredentialProvision(req prov.CredentialRequest) (prov.RequestStatus, prov.Credential) {
	p.log.Info("provisioning credentials")
	rs := prov.NewRequestStatusBuilder()

	appName := req.GetApplicationName()
	if appName == "" {
		return p.failed(rs, notFound("appName")), nil
	}

	appID := req.GetApplicationDetailsValue(common.AppID)
	if appID == "" {
		return p.failed(rs, notFound(appID)), nil
	}

	app, err := p.client.GetApp(appID)
	if err != nil {
		return p.failed(rs, fmt.Errorf("failed to retrieve app: %s", err)), nil
	}

	cr := prov.NewCredentialBuilder().SetCredential(map[string]interface{}{
		common.ClientID:     app.ClientID,
		common.ClientSecret: app.ClientSecret,
	})

	p.log.Info("created credentials")

	return rs.Success(), cr
}

func (p provisioner) failed(rs prov.RequestStatusBuilder, err error) prov.RequestStatus {
	rs.SetMessage(err.Error())
	p.log.Error(err)
	return rs.Failed()
}

func notFound(msg string) error {
	return fmt.Errorf("%s not found", msg)
}
