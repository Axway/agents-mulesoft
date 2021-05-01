package discovery

import (
	"time"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

// discovery implements the Repeater interface. Polls mulesoft for APIs.
type discovery struct {
	apiChan           chan *ServiceDetail
	assetCache        cache.Cache
	client            anypoint.ListAssetClient
	discoveryPageSize int
	pollInterval      time.Duration
	stopDiscovery     chan bool
	serviceHandler    ServiceHandler
}

func (a *discovery) Stop() {
	a.stopDiscovery <- true
}

func (a *discovery) OnConfigChange(cfg *config.MulesoftConfig) {
	a.pollInterval = cfg.PollInterval
	a.serviceHandler.OnConfigChange(cfg)
}

// Loop Discovery event loop.
func (a *discovery) Loop() {
	go func() {
		// Instant fist "tick"
		a.discoverAPIs()
		logrus.Info("Starting poller for Mulesoft APIs")
		ticker := time.NewTicker(a.pollInterval)
		for {
			select {
			case <-ticker.C:
				a.discoverAPIs()
				break
			case <-a.stopDiscovery:
				log.Debug("stopping discovery loop")
				ticker.Stop()
				break
			}
		}
	}()
}

// discoverAPIs Finds the APIs that are publishable.
func (a *discovery) discoverAPIs() {
	offset := 0
	pageSize := a.discoveryPageSize

	for {
		page := &anypoint.Page{Offset: offset, PageSize: pageSize}

		assets, err := a.client.ListAssets(page)
		if err != nil {
			log.Error(err)
		}

		for _, asset := range assets {
			svcDetails := a.serviceHandler.ToServiceDetails(&asset)
			if svcDetails != nil {
				for _, svc := range svcDetails {
					a.apiChan <- svc
				}
			}
		}

		if len(assets) != pageSize {
			break
		} else {
			offset += pageSize
		}
	}

	// Replacing asset cache rather than updating it
	freshAssetCache := cache.New()
	a.assetCache = freshAssetCache
}
