package traceability

import (
	"context"

	agentCache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	xAgentDetailClientID = "clientId"
)

type credentialHandler struct {
	credentialCache cache.Cache
}

// NewCredentialHandler creates a Handler for Credential and initializes credential cache with
// items from agent watch resource cache
func NewCredentialHandler(credentialCache cache.Cache, agentCacheManager agentCache.Manager) handler.Handler {
	h := &credentialHandler{
		credentialCache: credentialCache,
	}

	h.initCredentialCache(agentCacheManager)
	return h
}

// initCredentialCache - initializes credential cache with items from agent watch resource cache
func (h *credentialHandler) initCredentialCache(agentCacheManager agentCache.Manager) {
	keys := agentCacheManager.GetWatchResourceCacheKeys(mv1.CredentialGVK().Group, mv1.CredentialGVK().Kind)
	for _, key := range keys {
		credential := agentCacheManager.GetWatchResourceByKey(key)
		clientID, _ := util.GetAgentDetailsValue(credential, xAgentDetailClientID)
		if clientID != "" {
			h.credentialCache.Set(clientID, credential)
		}
	}
}

// Handle processes grpc events triggered for Credential
func (h *credentialHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := handler.GetActionFromContext(ctx)
	if resource.Kind != mv1.CredentialGVK().Kind {
		return nil
	}

	clientID, _ := util.GetAgentDetailsValue(resource, xAgentDetailClientID)
	if clientID != "" {
		if action == proto.Event_DELETED {
			h.credentialCache.Delete(clientID)
		} else {
			h.credentialCache.Set(clientID, resource)
		}
	}

	return nil
}
