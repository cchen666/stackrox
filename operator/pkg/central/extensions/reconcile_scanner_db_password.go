package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/renderer"
	"k8s.io/client-go/kubernetes"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	scannerDBPasswordKey = `password`
)

// ReconcileScannerDBPasswordExtension returns an extension that takes care of creating the scanner-db-password
// secret ahead of time.
func ReconcileScannerDBPasswordExtension(k8sClient kubernetes.Interface) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerDBPassword, k8sClient, nil)
}

func reconcileScannerDBPassword(ctx context.Context, c *platform.Central, k8sClient kubernetes.Interface, _ ctrlClient.Client, _ func(updateStatusFunc), log logr.Logger) error {
	run := &reconcileScannerDBPasswordExtensionRun{
		secretReconciliationExtension: secretReconciliationExtension{
			ctx:        ctx,
			k8sClient:  k8sClient,
			centralObj: c,
		},
	}
	return run.Execute()
}

type reconcileScannerDBPasswordExtensionRun struct {
	secretReconciliationExtension
}

func (r *reconcileScannerDBPasswordExtensionRun) Execute() error {
	// Delete any scanner-db password only if the CR is being deleted, or scanner is not enabled.
	shouldDelete := r.centralObj.DeletionTimestamp != nil || !r.centralObj.Spec.Scanner.IsEnabled()

	if err := r.reconcileSecret("scanner-db-password", !shouldDelete, r.validateScannerDBPasswordData, r.generateScannerDBPasswordData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner-db-password secret")
	}

	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) validateScannerDBPasswordData(data secretDataMap, _ bool) error {
	if len(data[scannerDBPasswordKey]) == 0 {
		return errors.Errorf("scanner-db-password secret must contain a non-empty %q entry", scannerDBPasswordKey)
	}
	return nil
}

func (r *reconcileScannerDBPasswordExtensionRun) generateScannerDBPasswordData() (secretDataMap, error) {
	data := secretDataMap{
		scannerDBPasswordKey: []byte(renderer.CreatePassword()),
	}
	return data, nil
}
