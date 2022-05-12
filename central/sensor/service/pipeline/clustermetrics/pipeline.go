package clustermetrics

import (
	"context"
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/node/globaldatastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), globaldatastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, nodes globaldatastore.GlobalDataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore: clusters,
		nodeStore:    nodes,
	}
}

type pipelineImpl struct {
	clusterStore clusterDataStore.DataStore
	nodeStore    globaldatastore.GlobalDataStore
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetClusterMetrics() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	fmt.Println("Sensor metrics: ", msg.GetClusterMetrics().CoreCapacity)
	return nil
}

func (p *pipelineImpl) OnFinish(_ string) {}
