package translation

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	central "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Translator translates and enriches helm values
type Translator struct {
	Config *rest.Config
}

// Translate translates and enriches helm values
func (t Translator) Translate(ctx context.Context, u *unstructured.Unstructured) (chartutil.Values, error) {
	c := central.Central{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-7251): make sure that the client we create here is kosher
	return Translate(ctx, kubernetes.NewForConfigOrDie(t.Config), c)
}

// Translate translates a Central CR into helm values.
func Translate(ctx context.Context, clientSet kubernetes.Interface, c central.Central) (chartutil.Values, error) {
	v := translation.NewValuesBuilder()

	v.AddAllFrom(translation.GetImagePullSecrets(c.Spec.ImagePullSecrets))
	v.AddAllFrom(getEnv(ctx, clientSet, c.Namespace, c.Spec.Egress))
	v.AddAllFrom(translation.GetTLSConfig(c.Spec.TLS))

	customize := translation.NewValuesBuilder()
	customize.AddAllFrom(translation.GetCustomize(c.Spec.Customize))

	if c.Spec.Central != nil {
		v.AddChild("central", getCentralComponentValues(ctx, clientSet, c.Namespace, c.Spec.Central))
		customize.AddChild("central", translation.GetCustomize(c.Spec.Central.Customize))
	}

	if c.Spec.Scanner != nil {
		v.AddChild("scanner", getScannerComponentValues(ctx, clientSet, c.Namespace, c.Spec.Scanner))
		if c.Spec.Scanner.Scanner != nil {
			customize.AddChild("scanner", translation.GetCustomize(c.Spec.Scanner.Scanner.Customize))
		}
		if c.Spec.Scanner.ScannerDB != nil {
			customize.AddChild("scanner-db", translation.GetCustomize(c.Spec.Scanner.ScannerDB.Customize))
		}
	}

	v.AddChild("customize", &customize)

	return v.Build()
}

func getEnv(ctx context.Context, clientSet kubernetes.Interface, namespace string, egress *central.Egress) *translation.ValuesBuilder {
	env := translation.NewValuesBuilder()
	if egress != nil {
		if egress.ConnectivityPolicy != nil {
			switch *egress.ConnectivityPolicy {
			case central.ConnectivityOnline:
				env.SetBoolValue("offlineMode", false)
			case central.ConnectivityOffline:
				env.SetBoolValue("offlineMode", true)
			default:
				return env.SetError(fmt.Errorf("invalid spec.egress.connectivityPolicy %q", *egress.ConnectivityPolicy))
			}
		}
		env.AddAllFrom(translation.NewBuilderFromSecret(ctx, clientSet, namespace, egress.ProxyConfigSecret, map[string]string{"config.yaml": "proxyConfig"}, "spec.egress.proxyConfigSecret"))
	}
	ret := translation.NewValuesBuilder()
	ret.AddChild("env", &env)
	return &ret
}

func getCentralComponentValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, c *central.CentralComponentSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()

	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	cv.AddAllFrom(translation.GetServiceTLS(ctx, clientSet, namespace, c.ServiceTLS, "spec.central.serviceTLS"))
	cv.SetStringMap("nodeSelector", c.NodeSelector)

	if c.TelemetryPolicy != nil {
		switch *c.TelemetryPolicy {
		case central.TelemetryEnabled:
			cv.SetBoolValue("disableTelemetry", false)
		case central.TelemetryDisabled:
			cv.SetBoolValue("disableTelemetry", true)
		default:
			return cv.SetError(fmt.Errorf("invalid spec.central.telemetryPolicy %q", *c.TelemetryPolicy))
		}
	}

	// TODO(ROX-7147): design CentralEndpointSpec, see central_types.go
	// TODO(ROX-7148): design CentralCryptoSpec, see central_types.go

	if c.AdminPasswordSecret != nil {
		cv.AddChild("adminPassword", translation.NewBuilderFromSecret(ctx, clientSet, namespace, c.AdminPasswordSecret, map[string]string{"value": "value"}, "spec.central.adminPasswordSecret"))
	} else {
		// TODO(ROX-7248): expose the autogenerated password
		return cv.SetError(errors.New("auto-generating admin password is not supported yet"))
	}

	if c.Persistence != nil {
		persistence := translation.NewValuesBuilder()
		persistence.SetString("hostPath", c.Persistence.HostPath)
		if c.Persistence.PersistentVolumeClaim != nil {
			pvc := translation.NewValuesBuilder()
			pvc.SetString("claimName", c.Persistence.PersistentVolumeClaim.ClaimName)
			if c.Persistence.PersistentVolumeClaim.CreateClaim != nil {
				switch *c.Persistence.PersistentVolumeClaim.CreateClaim {
				case central.ClaimCreate:
					pvc.SetBoolValue("createClaim", true)
				case central.ClaimReuse:
					pvc.SetBoolValue("createClaim", false)
				default:
					return cv.SetError(fmt.Errorf("invalid spec.central.persistence.persistentVolumeClaim.createClaim %q", *c.Persistence.PersistentVolumeClaim.CreateClaim))
				}
			}
			persistence.AddChild("persistentVolumeClaim", &pvc)
		}
		cv.AddChild("persistence", &persistence)
	}

	if c.Exposure != nil {
		exposure := translation.NewValuesBuilder()
		if c.Exposure.LoadBalancer != nil {
			lb := translation.NewValuesBuilder()
			lb.SetBool("enabled", c.Exposure.LoadBalancer.Enabled)
			lb.SetInt32("port", c.Exposure.LoadBalancer.Port)
			lb.SetString("ip", c.Exposure.LoadBalancer.IP)
			exposure.AddChild("loadBalancer", &lb)
		}
		if c.Exposure.NodePort != nil {
			np := translation.NewValuesBuilder()
			np.SetBool("enabled", c.Exposure.NodePort.Enabled)
			np.SetInt32("port", c.Exposure.NodePort.Port)
			exposure.AddChild("nodePort", &np)
		}
		if c.Exposure.Route != nil {
			route := translation.NewValuesBuilder()
			route.SetBool("enabled", c.Exposure.Route.Enabled)
			exposure.AddChild("route", &route)
		}
		cv.AddChild("exposure", &exposure)
	}
	return &cv
}

func getScannerComponentValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, s *central.ScannerComponentSpec) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()

	if s.ScannerComponent != nil {
		switch *s.ScannerComponent {
		case central.ScannerComponentDisabled:
			sv.SetBoolValue("disable", true)
		case central.ScannerComponentEnabled:
			sv.SetBoolValue("disable", false)
		default:
			return sv.SetError(fmt.Errorf("invalid spec.scanner.scannerComponent %q", *s.ScannerComponent))
		}
	}

	if s.Replicas != nil {
		sv.SetInt32("replicas", s.Replicas.Replicas)
		autoscaling := translation.NewValuesBuilder()
		if s.Replicas.AutoScaling != nil {
			switch *s.Replicas.AutoScaling {
			case central.ScannerAutoScalingDisabled:
				autoscaling.SetBoolValue("disable", true)
			case central.ScannerAutoScalingEnabled:
				autoscaling.SetBoolValue("disable", false)
			default:
				return autoscaling.SetError(fmt.Errorf("invalid spec.scanner.replicas.autoScaling %q", *s.Replicas.AutoScaling))
			}
		}
		autoscaling.SetInt32("minReplicas", s.Replicas.MinReplicas)
		autoscaling.SetInt32("maxReplicas", s.Replicas.MaxReplicas)
		sv.AddChild("autoscaling", &autoscaling)
	}

	if s.Logging != nil {
		sv.SetString("logLevel", s.Logging.Level)
	}

	if s.Scanner != nil {
		sv.AddAllFrom(translation.GetServiceTLS(ctx, clientSet, namespace, s.Scanner.ServiceTLS, "spec.scanner.scanner.serviceTLS"))
		sv.SetStringMap("nodeSelector", s.Scanner.NodeSelector)
		sv.AddChild(translation.ResourcesKey, translation.GetResources(s.Scanner.Resources))
	}

	if s.ScannerDB != nil {
		sv.AddAllFrom(translation.GetServiceTLSWithKey(ctx, clientSet, namespace, s.ScannerDB.ServiceTLS, "spec.scanner.scannerDB.serviceTLS", "dbServiceTLS"))
		sv.SetStringMap("dbNodeSelector", s.ScannerDB.NodeSelector)
		sv.AddChild("dbResources", translation.GetResources(s.ScannerDB.Resources))
	}

	return &sv
}
