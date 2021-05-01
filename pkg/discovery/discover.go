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

func (d *discovery) Stop() {
	d.stopDiscovery <- true
}

func (d *discovery) OnConfigChange(cfg *config.MulesoftConfig) {
	d.pollInterval = cfg.PollInterval
	d.serviceHandler.OnConfigChange(cfg)
}

// Loop Discovery event loop.
func (d *discovery) Loop() {
	go func() {
		// Instant fist "tick"
		d.discoverAPIs()
		logrus.Info("Starting poller for Mulesoft APIs")
		ticker := time.NewTicker(d.pollInterval)
		for {
			select {
			case <-ticker.C:
				d.discoverAPIs()
				break
			case <-d.stopDiscovery:
				log.Debug("stopping discovery loop")
				ticker.Stop()
				break
			}
		}
	}()
}

// discoverAPIs Finds the APIs that are publishable.
func (d *discovery) discoverAPIs() {
	offset := 0
	pageSize := d.discoveryPageSize

	for {
		page := &anypoint.Page{Offset: offset, PageSize: pageSize}

		assets, err := d.client.ListAssets(page)
		if err != nil {
			log.Error(err)
		}

		for _, asset := range assets {
			svcDetails := d.serviceHandler.ToServiceDetails(&asset)
			if svcDetails != nil {
				for _, svc := range svcDetails {
					d.apiChan <- svc
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
	d.assetCache = freshAssetCache
}
