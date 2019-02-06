package manager

import (
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageIntegrationDS "github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/metrics"
	multiplierDS "github.com/stackrox/rox/central/multiplier/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
)

var log = logging.LoggerForModule()

// Manager manages changes to the risk of the deployments
type Manager interface {
	UpdateMultiplier(multiplier *storage.Multiplier)
	RemoveMultiplier(id string)

	ReprocessRisk()
	ReprocessDeploymentRisk(deployment *storage.Deployment)
}

type managerImpl struct {
	deploymentStorage       deploymentDS.DataStore
	imageIntegrationStorage imageIntegrationDS.DataStore
	multiplierStorage       multiplierDS.Store

	imageEnricher enricher.ImageEnricher
	scorer        risk.Scorer
}

// New returns a new manager
func New(deploymentStorage deploymentDS.DataStore,
	multiplierStorage multiplierDS.Store,
	scorer risk.Scorer) (Manager, error) {
	m := &managerImpl{
		deploymentStorage: deploymentStorage,
		multiplierStorage: multiplierStorage,
		scorer:            scorer,
	}
	if err := m.initializeMultipliers(); err != nil {
		return nil, err
	}
	return m, nil
}

func (e *managerImpl) initializeMultipliers() error {
	protoMultipliers, err := e.multiplierStorage.GetMultipliers()
	if err != nil {
		return err
	}
	for _, mult := range protoMultipliers {
		e.scorer.UpdateUserDefinedMultiplier(mult)
	}
	return nil
}

// UpdateMultiplier upserts a multiplier into the scorer
func (e *managerImpl) UpdateMultiplier(multiplier *storage.Multiplier) {
	e.scorer.UpdateUserDefinedMultiplier(multiplier)
	e.ReprocessRisk()
}

// RemoveMultiplier removes a multiplier from the scorer
func (e *managerImpl) RemoveMultiplier(id string) {
	e.scorer.RemoveUserDefinedMultiplier(id)
	e.ReprocessRisk()
}

// ReprocessRisk iterates over all of the deployments and reprocesses the risk for them
func (e *managerImpl) ReprocessRisk() {
	deployments, err := e.deploymentStorage.GetDeployments()
	if err != nil {
		log.Errorf("Error reprocessing risk: %s", err)
		return
	}

	for _, deployment := range deployments {
		if err := e.addRiskToDeployment(deployment); err != nil {
			log.Errorf("Error reprocessing deployment risk: %s", err)
			return
		}
	}
}

// ReprocessDeploymentRisk will reprocess the passed deployments risk and save the results
func (e *managerImpl) ReprocessDeploymentRisk(deployment *storage.Deployment) {
	defer metrics.ObserveRiskProcessingDuration(time.Now())
	deployment = protoutils.CloneStorageDeployment(deployment)
	if err := e.addRiskToDeployment(deployment); err != nil {
		log.Errorf("Error reprocessing risk for deployment %s: %s", deployment.GetName(), err)
	}
}

// addRiskToDeployment will add the risk
func (e *managerImpl) addRiskToDeployment(deployment *storage.Deployment) error {
	deployment.Risk = e.scorer.Score(deployment)
	return e.deploymentStorage.UpdateDeployment(deployment)
}
