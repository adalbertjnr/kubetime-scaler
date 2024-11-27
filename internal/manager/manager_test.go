package manager

import (
	"context"
	"testing"
	"time"

	downscalergov1alpha1 "github.com/adalbertjnr/downscalerk8s/api/v1alpha1"
	"github.com/adalbertjnr/downscalerk8s/internal/client"
	"github.com/adalbertjnr/downscalerk8s/internal/factory"
	objecttypes "github.com/adalbertjnr/downscalerk8s/internal/types"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupDownscalerObject() downscalergov1alpha1.Downscaler {
	return downscalergov1alpha1.Downscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "downscaler.go/v1alpha1",
			Kind:       "Downscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "downscaler-test",
			Namespace: "downscaler-ns-test",
		},
		Spec: downscalergov1alpha1.DownscalerSpec{
			Config:   downscalergov1alpha1.Config{CronLoggerInterval: 60},
			Schedule: downscalergov1alpha1.Schedule{TimeZone: "America/Sao_Paulo", Recurrence: "@daily"},
			DownscalerOptions: downscalergov1alpha1.DownscalerOptions{
				ResourceScaling: []objecttypes.ResourceType{"deployments", "statefulset"},
				TimeRules: &downscalergov1alpha1.TimeRules{
					Rules: []downscalergov1alpha1.Rules{
						{
							Name:            "Rule to test",
							Namespaces:      []downscalergov1alpha1.Namespace{"test-app"},
							DownscaleTime:   "fake",
							UpscaleTime:     "fake",
							OverrideScaling: []objecttypes.ResourceType{"deployments", "statefulset"},
						},
					},
				},
			},
		},
	}
}

func TestScaleDeployment(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-app",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(5),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testDeployment).Build()

	c := client.NewAPIClient(fakeClient)

	downscalerObject := setupDownscalerObject()

	now := time.Now()
	futureDownscale := now.Add(time.Minute).Format("15:04")
	futureUpscale := now.Add(time.Minute).Format("15:04")

	downscalerObject.Spec.DownscalerOptions.TimeRules.Rules[0].UpscaleTime = futureUpscale
	downscalerObject.Spec.DownscalerOptions.TimeRules.Rules[0].DownscaleTime = futureDownscale

	dm := (&Downscaler{}).
		Client(c).
		Factory(factory.NewScalerFactory(c, nil, logr.Logger{})).
		Persistence(nil).
		Add(context.Background(), downscalerObject).
		Logger(logr.Logger{})

	loc, _ := time.LoadLocation(downscalerObject.Spec.Schedule.TimeZone)

	dm.cron = cron.New(cron.WithLocation(loc))
	defer dm.cron.Stop()

	dm.initializeCronTasks()

	time.Sleep(time.Second * 80)

	updatedDeployment := &appsv1.Deployment{}
	err := fakeClient.Get(context.TODO(), types.NamespacedName{Name: "test-app", Namespace: "test-app"}, updatedDeployment)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), *updatedDeployment.Spec.Replicas)
}

func int32Ptr(i int32) *int32 {
	return &i
}
