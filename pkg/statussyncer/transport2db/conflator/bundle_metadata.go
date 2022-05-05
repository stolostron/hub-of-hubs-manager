package conflator

import (
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/transport"
	"github.com/stolostron/hub-of-hubs-data-types/bundle/status"
)

// BundleMetadata abstracts metadata of conflation elements inside the conflation units.
type BundleMetadata struct {
	bundleType    string
	bundleVersion *status.BundleVersion
	// transport metadata is information we need for marking bundle as consumed in transport (e.g. commit offset)
	transportBundleMetadata transport.BundleMetadata
}
