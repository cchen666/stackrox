package clustermetrics

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultInterval = 2 * time.Minute
)

var log = logging.LoggerForModule()

// ClusterMetrics collects metrics from secured clusters and sends them to Central.
//go:generate mockgen-wrapper
type ClusterMetrics interface {
	common.SensorComponent
}

func New(k8sClient kubernetes.Interface) ClusterMetrics {
	return &clusterMetricsImpl{
		output:          make(chan *central.MsgFromSensor),
		stopper:         concurrency.NewStopper(),
		pollingInterval: defaultInterval,
		k8sClient:       k8sClient,
	}
}

type clusterMetricsImpl struct {
	output          chan *central.MsgFromSensor
	stopper         concurrency.Stopper
	pollingInterval time.Duration
	k8sClient       kubernetes.Interface
}

func (cm *clusterMetricsImpl) Start() error {
	go cm.Poll()
	return nil
}

func (cm *clusterMetricsImpl) Stop(err error) {
	cm.stopper.Stop()
	cm.stopper.WaitForStopped()
}

func (cm *clusterMetricsImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (cm *clusterMetricsImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (cm *clusterMetricsImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return cm.output
}

func (cm *clusterMetricsImpl) ProcessIndicator(pi *storage.ProcessIndicator) {
}

func (cm *clusterMetricsImpl) Poll() {
	defer cm.stopper.Stopped()

	ticker := time.NewTicker(cm.pollingInterval)
	go func() {
		for {
			select {
			case <-cm.stopper.StopDone():
				return
			case <-ticker.C:
				if metrics, err := cm.collectMetrics(); err != nil {
					cm.output <- &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_ClusterMetrics{
							ClusterMetrics: metrics,
						},
					}
				} else {
					log.Errorf("Collection of cluster metrics failed: %v", err.Error())
				}
			}
		}
	}()
}

func (cm *clusterMetricsImpl) collectMetrics() (*central.ClusterMetrics, error) {
	nodes, err := cm.k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var capacity int64 = 0
	for _, node := range nodes.Items {
		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			capacity += cpu.Value()
		}
	}
	return &central.ClusterMetrics{CoreCapacity: capacity}, nil
}
