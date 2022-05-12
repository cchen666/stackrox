package clustermetrics

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// Max time to receive health info status. You may want to increase it if you plan to step through the code with debugger.
	updateTimeout = 3 * time.Second
	// How frequently should updater provide health info during tests.
	updateInterval = 1 * time.Millisecond
	// Environment variable to hold pod namespace. In actual k8s deployment it is set by helm/yaml file.
	namespaceVar = "POD_NAMESPACE"
)

func TestClusterMetrics(t *testing.T) {
	suite.Run(t, new(ClusterMetricsTestSuite))
}

type ClusterMetricsTestSuite struct {
	suite.Suite

	client *fake.Clientset
	env    *envisolator.EnvIsolator
}

type expectedHealthInfo struct {
	version               string
	desired, ready, nodes int32
	errors                []string
}

func (s *ClusterMetricsTestSuite) SetupTest() {
	s.client = fake.NewSimpleClientset()
	s.env = envisolator.NewEnvIsolator(s.T())
	s.env.Setenv(namespaceVar, "stackrox-mock-ns")
}

func (s *ClusterMetricsTestSuite) TearDownTest() {
	s.env.RestoreAll()
}

func (s *ClusterMetricsTestSuite) TestHappyCase() {
	ds := makeDaemonSet()
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *ClusterMetricsTestSuite) TestSlimSuffixTrimmed() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "mock/image:v5.0.1fat-slim"
	s.addDaemonSet(ds)

	health := s.getHealthInfo(1)

	s.assertVersion(health, "v5.0.1fat")
}

func (s *ClusterMetricsTestSuite) TestLatestSuffixTrimmed() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "mock/image:v5.0.2old-latest"
	s.addDaemonSet(ds)

	health := s.getHealthInfo(1)

	s.assertVersion(health, "v5.0.2old")
}

func (s *ClusterMetricsTestSuite) TestDaemonSetWithoutContainerSpec() {
	ds := makeDaemonSet()
	ds.Spec = appsV1.DaemonSetSpec{} // Erase containers information.
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "", desired: 6, ready: 4, nodes: 7, errors: []string{"collector version"},
	})
}

func (s *ClusterMetricsTestSuite) TestWithoutDaemonSet() {
	// No DaemonSet added.
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "", desired: -1, ready: -1, nodes: 7, errors: []string{"collector DaemonSet"},
	})
}

func (s *ClusterMetricsTestSuite) TestWithoutNodes() {
	ds := makeDaemonSet()
	s.addDaemonSet(ds)
	// No nodes get added.

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 0, errors: nil,
	})
}

func (s *ClusterMetricsTestSuite) TestVersionWithoutTag() {
	ds := makeDaemonSet()
	ds.Spec.Template.Spec.Containers[0].Image = "blah/without/tags"
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "blah/without/tags", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *ClusterMetricsTestSuite) TestCanSendMultipleUpdates() {
	s.addDaemonSet(makeDaemonSet())
	s.addNodes(7)

	health := s.getHealthInfo(5)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *ClusterMetricsTestSuite) TestCustomNamespaceHappyCase() {
	const customNs = "custom-test-ns"
	s.env.Setenv(namespaceVar, customNs)

	ds := makeDaemonSet()
	ds.ObjectMeta.Namespace = customNs
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *ClusterMetricsTestSuite) TestNamespaceFallback() {
	s.env.Unsetenv(namespaceVar)

	ds := makeDaemonSet()
	ds.ObjectMeta.Namespace = namespaces.StackRox
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "v456", desired: 6, ready: 4, nodes: 7, errors: nil,
	})
}

func (s *ClusterMetricsTestSuite) TestNamespaceMismatch() {
	s.env.Setenv(namespaceVar, "where-things-should-be")

	ds := makeDaemonSet()
	ds.ObjectMeta.Namespace = "where-things-are"
	s.addDaemonSet(ds)
	s.addNodes(7)

	health := s.getHealthInfo(1)

	s.assertHealthInfo(health, expectedHealthInfo{
		version: "", desired: -1, ready: -1, nodes: 7, errors: []string{"unable to find collector DaemonSet in namespace \"where-things-should-be\""},
	})
}

func (s *ClusterMetricsTestSuite) getHealthInfo(times int) *storage.CollectorHealthInfo {
	timer := time.NewTimer(updateTimeout)
	updater := NewUpdater(s.client, updateInterval)

	err := updater.Start()
	s.Require().NoError(err)
	defer updater.Stop(nil)

	var healthInfo *storage.CollectorHealthInfo

	for i := 0; i < times; i++ {
		select {
		case response := <-updater.ResponsesC():
			healthInfo = response.Msg.(*central.MsgFromSensor_ClusterHealthInfo).ClusterHealthInfo.CollectorHealthInfo
		case <-timer.C:
			s.Fail("Timed out while waiting for cluster health update")
		}
	}

	return healthInfo
}

func makeDaemonSet() appsV1.DaemonSet {
	return appsV1.DaemonSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "collector",
			Namespace: "stackrox-mock-ns",
		},
		Spec: appsV1.DaemonSetSpec{
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{Name: "collector", Image: "mock/image:v456"},
					},
				},
			},
		},
		Status: appsV1.DaemonSetStatus{
			DesiredNumberScheduled: 6,
			NumberReady:            4,
		},
	}
}

func (s *ClusterMetricsTestSuite) addDaemonSet(ds appsV1.DaemonSet) {
	_, err := s.client.AppsV1().DaemonSets(ds.ObjectMeta.Namespace).Create(context.Background(), &ds, metaV1.CreateOptions{})
	s.Require().NoError(err)
}

func (s *ClusterMetricsTestSuite) addNodes(count int) {
	for i := 0; i < count; i++ {
		_, err := s.client.CoreV1().Nodes().Create(context.Background(), &coreV1.Node{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "mock-node-" + strconv.Itoa(i),
			},
		}, metaV1.CreateOptions{})
		s.Require().NoError(err)
	}
}

func (s *ClusterMetricsTestSuite) assertHealthInfo(actual *storage.CollectorHealthInfo, expected expectedHealthInfo) {
	s.assertVersion(actual, expected.version)
	s.assertTotalDesiredPods(actual, expected.desired)
	s.assertTotalReadyPods(actual, expected.ready)
	s.assertTotalRegisteredNodes(actual, expected.nodes)
	s.assertStatusErrors(actual, expected.errors...)
}

func (s *ClusterMetricsTestSuite) assertVersion(health *storage.CollectorHealthInfo, expected string) {
	s.Equal(expected, health.Version)
}

func (s *ClusterMetricsTestSuite) assertTotalDesiredPods(health *storage.CollectorHealthInfo, expected int32) {
	var actual int32
	switch v := health.TotalDesiredPodsOpt.(type) {
	case *storage.CollectorHealthInfo_TotalDesiredPods:
		actual = v.TotalDesiredPods
	case nil:
		actual = -1
	default:
		s.FailNowf("Unexpected total desired pods value type", "actual value: %#v", health.TotalDesiredPodsOpt)
	}
	s.Equalf(expected, actual, "Unexpected value of total desired pods %#v", health.TotalDesiredPodsOpt)
}

func (s *ClusterMetricsTestSuite) assertTotalReadyPods(health *storage.CollectorHealthInfo, expected int32) {
	var actual int32
	switch v := health.TotalReadyPodsOpt.(type) {
	case *storage.CollectorHealthInfo_TotalReadyPods:
		actual = v.TotalReadyPods
	case nil:
		actual = -1
	default:
		s.FailNowf("Unexpected total ready pods value type", "actual value: %#v", health.TotalReadyPodsOpt)
	}
	s.Equalf(expected, actual, "Unexpected value of total ready pods %#v", health.TotalReadyPodsOpt)
}

func (s *ClusterMetricsTestSuite) assertTotalRegisteredNodes(health *storage.CollectorHealthInfo, expected int32) {
	var actual int32
	switch v := health.TotalRegisteredNodesOpt.(type) {
	case *storage.CollectorHealthInfo_TotalRegisteredNodes:
		actual = v.TotalRegisteredNodes
	case nil:
		actual = -1
	default:
		s.FailNowf("Unexpected total registered nodes value type", "actual value: %#v", health.TotalRegisteredNodesOpt)
	}
	s.Equalf(expected, actual, "Unexpected value of total registered nodes %#v", health.TotalReadyPodsOpt)
}

func (s *ClusterMetricsTestSuite) assertStatusErrors(health *storage.CollectorHealthInfo, expected ...string) {
	s.Len(health.StatusErrors, len(expected))
	for _, e := range expected {
		var found int
		for _, s := range health.StatusErrors {
			if strings.Contains(s, e) {
				found++
			}
		}
		if found != 1 {
			s.Failf(
				"Did not find expected error",
				"Expected to find exactly 1 substring %#v in %#v, found %d",
				e,
				health.StatusErrors,
				found)
		}
	}
}

