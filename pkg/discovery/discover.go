package discovery

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util"

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
			go func(asset anypoint.Asset) {
				svcDetails := d.serviceHandler.ToServiceDetails(&asset)
				if svcDetails != nil {
					for _, svc := range svcDetails {
						d.apiChan <- svc
					}
				}
			}(asset)
		}

		if len(assets) != pageSize {
			break
		} else {
			offset += pageSize
		}
	}
}

// getRevisions add revisions to the cache when the agent starts so that apis can be checked against what is saved in central when discovery starts.
func (d *discovery) getRevisions() {
	revs, err := d.centralClient.GetAPIRevisions(map[string]string{}, "")
	if err != nil {
		logrus.Errorf("failed to populate cache with revisions: %s", err)
		return
	}

	for _, rev := range revs {
		apiID, _ := util.GetAgentDetailsValue(rev, common.AttrAPIID)
		productVersion, _ := util.GetAgentDetailsValue(rev, common.AttrProductVersion)
		checksum, _ := util.GetAgentDetailsValue(rev, common.AttrChecksum)
		if apiID == "" || productVersion == "" || checksum == "" {
			log.Errorf("failed to save revision to the cache. apiID: '%s'. product version: '%s', checksum: '%s'", apiID, productVersion, checksum)
			continue
		}

		secondaryKey := common.FormatAPICacheKey(util.ToString(apiID), util.ToString(productVersion))
		err = d.cache.SetWithSecondaryKey(util.ToString(checksum), secondaryKey, rev)
		if err != nil {
			logrus.WithError(err).Error("failed to save to the cache")
		}
	}
}
