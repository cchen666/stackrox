package service

import (
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	runTimeDetectiomn "github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		datastore.Singleton(),
		processor.Singleton(),
		buildTimeDetection.SingletonPolicySet(),
		deployTimeDetection.SingletonPolicySet(),
		runTimeDetectiomn.SingletonPolicySet(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
