package discovery

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agents-mulesoft/pkg/common"

	"github.com/Axway/agents-mulesoft/pkg/config"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agents-mulesoft/pkg/anypoint"
)

// discovery implements the Repeater interface. Polls mulesoft for APIs.
type discovery struct {
	apiChan           chan *ServiceDetail
	cache             cache.Cache
	centralClient     apic.Client
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
		d.getRevisions()
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

// discoverAPIs Finds APIs from exchange
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
}

// getRevisions add revisions to the cache when the agent starts.
func (d *discovery) getRevisions() {
	revs, err := d.centralClient.GetAPIRevisions(map[string]string{}, "")
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, rev := range revs {
		secondaryKey := common.FormatAPICacheKey(rev.Attributes[common.AttrAPIID], rev.Attributes[common.AttrProductVersion])
		err := d.cache.SetWithSecondaryKey(rev.Attributes[common.AttrChecksum], secondaryKey, rev)
		if err != nil {
			logrus.WithError(err).Error("failed to save to the cache")
		}
	}
}
