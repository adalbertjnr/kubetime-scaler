package manager

import (
	"context"
	"testing"
	"time"

	downscalergov1alpha1 "github.com/adalbertjnr/downscalerk8s/api/v1alpha1"
	apiclient "github.com/adalbertjnr/downscalerk8s/internal/client"
	"github.com/adalbertjnr/downscalerk8s/internal/factory"
	objecttypes "github.com/adalbertjnr/downscalerk8s/internal/types"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	defaultFormatTime = "15:04:05"
	oneSecond         = time.Second + 50*time.Millisecond
)

type testCase struct {
	name             string
	test             string
	objectName       []string
	namespaces       []downscalergov1alpha1.Namespace
	overrideReplicas []objecttypes.ResourceType
	initiaReplicas   int32
	expectedReplicas int32
}

func setupDownscalerInstance(c *apiclient.APIClient, downscalerObject downscalergov1alpha1.Downscaler) *Downscaler {
	return (&Downscaler{}).
		Client(c).
		Factory(factory.NewScalerFactory(c, nil, logr.Logger{})).
		Persistence(nil).
		Add(context.Background(), downscalerObject).
		Logger(logr.Logger{})
}

func setupDownscalerObject(downscaleTime, upscaleTime string, ruleNameDescription string, namespaces []downscalergov1alpha1.Namespace, resourceTypes []objecttypes.ResourceType) downscalergov1alpha1.Downscaler {
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
							Name:            ruleNameDescription,
							Namespaces:      namespaces,
							DownscaleTime:   downscaleTime,
							UpscaleTime:     upscaleTime,
							OverrideScaling: resourceTypes,
						},
					},
				},
			},
		},
	}
}

func TestDownscalingDeployments(t *testing.T) {

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	testCases := []testCase{
		{
			test:             "TestSingleNamespaceSingleDeployment",
			name:             "single namespace. downscale deployment deployment1",
			objectName:       []string{"deployment1"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app1"},
			initiaReplicas:   5,
			expectedReplicas: 0,
		},
		{
			test:             "TestMultipleNamespacesMultipleDeployments",
			name:             "multiple namespaces and deployments. downscale deployments deployment2,deployment3",
			objectName:       []string{"deployment2", "deployment3"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app2", "ns-app3"},
			initiaReplicas:   5,
			expectedReplicas: 0,
		},
		{
			test:             "TestSingleNamespaceSingleDeploymentWithStatefulOverride",
			name:             "single namespace and deployment. downscale deployment deployment4 - should only try to downscale statefulset which means the deployment will be with 5 replicas",
			objectName:       []string{"deployment4"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app4"},
			overrideReplicas: []objecttypes.ResourceType{"statefulset"},
			initiaReplicas:   5,
			expectedReplicas: 5,
		},
		{
			test:             "TestMultipleNamespacesMultipleDeploymentsWithStatefulOverride",
			name:             "multiple namespaces and deployments. downscale deployments deployment5,deployment6 - should only try to downscale statefulset which means the deployments will be with 5 replicas",
			objectName:       []string{"deployment5", "deployment6"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app5", "ns-app6"},
			overrideReplicas: []objecttypes.ResourceType{"statefulset"},
			initiaReplicas:   5,
			expectedReplicas: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			var clientObjectList []client.Object
			for i := range tc.objectName {
				clientObject := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.objectName[i],
						Namespace: tc.namespaces[i].String(),
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &tc.initiaReplicas,
					},
				}
				clientObjectList = append(clientObjectList, clientObject)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(clientObjectList...).Build()
			c := apiclient.NewAPIClient(fakeClient)

			now := time.Now()
			testDownscaleTime := now.Add(time.Second).Format(defaultFormatTime)
			downscalerObject := setupDownscalerObject(testDownscaleTime, "", tc.name, tc.namespaces, tc.overrideReplicas)

			location, err := time.LoadLocation(downscalerObject.Spec.Schedule.TimeZone)
			if err != nil {
				t.Fatalf("error loading timezone location: %v", err)
			}

			dm := setupDownscalerInstance(c, downscalerObject)
			dm.cron = cron.New(cron.WithLocation(location), cron.WithSeconds())
			defer dm.cron.Stop()

			dm.initializeCronTasks()

			<-time.After(oneSecond)

			for i := range tc.objectName {
				updatedObject := &appsv1.Deployment{}
				if err := c.Get(tc.namespaces[i].String(), updatedObject, tc.objectName[i]); err != nil {
					t.Fatalf("error getting updated deployment: %v", err)
				}
				assert.Equal(t, tc.expectedReplicas, *updatedObject.Spec.Replicas)
			}
		})
	}
}

func TestDownscalingStatefulsets(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	testCases := []testCase{
		{
			test:             "TestSingleNamespaceSingleStatefulset",
			name:             "single namespace. downscale statefulset statefulset1",
			objectName:       []string{"statefulset1"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app1"},
			initiaReplicas:   2,
			expectedReplicas: 0,
		},
		{
			test:             "TestMultipleNamespacesMultipleStatefulsets",
			name:             "multiple namespaces and statefulsets. downscale statefulsets statefulset2,statefulset3",
			objectName:       []string{"statefulset2", "statefulset3"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app2", "ns-app3"},
			initiaReplicas:   2,
			expectedReplicas: 0,
		},
		{
			test:             "TestSingleNamespaceSingleStatefulsetWithStatefulOverride",
			name:             "single namespace and statefulset. downscale statefulset statefulset4 - should only try to downscale deployments which means the statefulset will be with 2 replicas",
			objectName:       []string{"statefulset4"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app4"},
			overrideReplicas: []objecttypes.ResourceType{"deployments"},
			initiaReplicas:   2,
			expectedReplicas: 2,
		},
		{
			test:             "TestMultipleNamespacesMultipleStatefulsetsWithStatefulOverride",
			name:             "multiple namespaces and statefulsets. downscale statefulsets statefulset5,statefulset6 - should only try to downscale deployments which means the statefulsets will be with 2 replicas",
			objectName:       []string{"statefulset5", "statefulset6"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app5", "ns-app6"},
			overrideReplicas: []objecttypes.ResourceType{"deployments"},
			initiaReplicas:   2,
			expectedReplicas: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			var clientObjectList []client.Object
			for i := range tc.objectName {
				clientObject := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.objectName[i],
						Namespace: tc.namespaces[i].String(),
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &tc.initiaReplicas,
					},
				}
				clientObjectList = append(clientObjectList, clientObject)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(clientObjectList...).Build()
			c := apiclient.NewAPIClient(fakeClient)

			now := time.Now()
			testDownscaleTime := now.Add(time.Second).Format(defaultFormatTime)
			downscalerObject := setupDownscalerObject(testDownscaleTime, "", tc.name, tc.namespaces, tc.overrideReplicas)

			location, err := time.LoadLocation(downscalerObject.Spec.Schedule.TimeZone)
			if err != nil {
				t.Fatalf("error loading timezone location: %v", err)
			}

			dm := setupDownscalerInstance(c, downscalerObject)
			dm.cron = cron.New(cron.WithLocation(location), cron.WithSeconds())
			defer dm.cron.Stop()

			dm.initializeCronTasks()

			<-time.After(oneSecond)

			for i := range tc.objectName {
				updatedObject := &appsv1.StatefulSet{}
				if err := c.Get(tc.namespaces[i].String(), updatedObject, tc.objectName[i]); err != nil {
					t.Fatalf("error getting updated deployment: %v", err)
				}
				assert.Equal(t, tc.expectedReplicas, *updatedObject.Spec.Replicas)
			}
		})
	}
}
func TestUpscalingDeployments(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	testCases := []testCase{
		{
			test:             "TestSingleNamespaceSingleDeployment",
			name:             "single namespace. upscale deployment deployment1",
			objectName:       []string{"deployment1"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app1"},
			initiaReplicas:   0,
			expectedReplicas: 1,
		},
		{
			test:             "TestMultipleNamespacesMultipleDeployments",
			name:             "multiple namespaces and deployments. upscale deployments deployment2,deployment3",
			objectName:       []string{"deployment2", "deployment3"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app2", "ns-app3"},
			initiaReplicas:   0,
			expectedReplicas: 1,
		},
		{
			test:             "TestSingleNamespaceSingleDeploymentWithStatefulOverride",
			name:             "single namespace and deployment. upscale deployment deployment4 - should only try to upscale statefulset which means the deployment will be with 0 replicas",
			objectName:       []string{"deployment4"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app4"},
			overrideReplicas: []objecttypes.ResourceType{"statefulset"},
			initiaReplicas:   0,
			expectedReplicas: 0,
		},
		{
			test:             "TestMultipleNamespacesMultipleDeploymentsWithStatefulOverride",
			name:             "multiple namespaces and deployments. upscale deployments deployment5,deployment6 - should only try to upscale statefulset which means the deployments will be with 0 replicas",
			objectName:       []string{"deployment5", "deployment6"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app5", "ns-app6"},
			overrideReplicas: []objecttypes.ResourceType{"statefulset"},
			initiaReplicas:   0,
			expectedReplicas: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			var clientObjectList []client.Object
			for i := range tc.objectName {
				clientObject := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.objectName[i],
						Namespace: tc.namespaces[i].String(),
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &tc.initiaReplicas,
					},
				}
				clientObjectList = append(clientObjectList, clientObject)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(clientObjectList...).Build()
			c := apiclient.NewAPIClient(fakeClient)

			now := time.Now()
			testUpscaleTime := now.Add(time.Second).Format(defaultFormatTime)
			downscalerObject := setupDownscalerObject("", testUpscaleTime, tc.name, tc.namespaces, tc.overrideReplicas)

			location, err := time.LoadLocation(downscalerObject.Spec.Schedule.TimeZone)
			if err != nil {
				t.Fatalf("error loading timezone location: %v", err)
			}

			dm := setupDownscalerInstance(c, downscalerObject)
			dm.cron = cron.New(cron.WithLocation(location), cron.WithSeconds())
			defer dm.cron.Stop()

			dm.initializeCronTasks()

			<-time.After(oneSecond)

			for i := range tc.objectName {
				updatedObject := &appsv1.Deployment{}
				if err := c.Get(tc.namespaces[i].String(), updatedObject, tc.objectName[i]); err != nil {
					t.Fatalf("error getting updated statefulset: %v", err)
				}
				assert.Equal(t, tc.expectedReplicas, *updatedObject.Spec.Replicas)
			}
		})
	}
}

func TestUpscalingStatefulsets(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	testCases := []testCase{
		{
			test:             "TestSingleNamespaceSingleStatefulset",
			name:             "single namespace. upscale statefulset statefulset1",
			objectName:       []string{"statefulset1"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app1"},
			initiaReplicas:   0,
			expectedReplicas: 1,
		},
		{
			test:             "TestMultipleNamespacesMultipleStatefulsets",
			name:             "multiple namespaces and statefulsets. upscale statefulsets statefulset2,statefulset3",
			objectName:       []string{"statefulset2", "statefulset3"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app2", "ns-app3"},
			initiaReplicas:   0,
			expectedReplicas: 1,
		},
		{
			test:             "TestSingleNamespaceSingleStatefulsetWithStatefulOverride",
			name:             "single namespace and statefulset. upscale statefulset statefulset4 - should only try to upscale deployments which means the statefulset will be with 0 replicas",
			objectName:       []string{"statefulset4"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app4"},
			overrideReplicas: []objecttypes.ResourceType{"deployments"},
			initiaReplicas:   0,
			expectedReplicas: 0,
		},
		{
			test:             "TestMultipleNamespacesMultipleStatefulsetsWithStatefulOverride",
			name:             "multiple namespaces and statefulsets. upscale statefulsets statefulset5,statefulset6 - should only try to upscale deployments which means the statefulsets will be with 0 replicas",
			objectName:       []string{"statefulset5", "statefulset6"},
			namespaces:       []downscalergov1alpha1.Namespace{"ns-app5", "ns-app6"},
			overrideReplicas: []objecttypes.ResourceType{"deployments"},
			initiaReplicas:   0,
			expectedReplicas: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			var clientObjectList []client.Object
			for i := range tc.objectName {
				clientObject := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.objectName[i],
						Namespace: tc.namespaces[i].String(),
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &tc.initiaReplicas,
					},
				}
				clientObjectList = append(clientObjectList, clientObject)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(clientObjectList...).Build()
			c := apiclient.NewAPIClient(fakeClient)

			now := time.Now()
			testUpscaleTime := now.Add(time.Second).Format(defaultFormatTime)
			downscalerObject := setupDownscalerObject("", testUpscaleTime, tc.name, tc.namespaces, tc.overrideReplicas)

			location, err := time.LoadLocation(downscalerObject.Spec.Schedule.TimeZone)
			if err != nil {
				t.Fatalf("error loading timezone location: %v", err)
			}

			dm := setupDownscalerInstance(c, downscalerObject)
			dm.cron = cron.New(cron.WithLocation(location), cron.WithSeconds())
			defer dm.cron.Stop()

			dm.initializeCronTasks()

			<-time.After(oneSecond)

			for i := range tc.objectName {
				updatedObject := &appsv1.StatefulSet{}
				if err := c.Get(tc.namespaces[i].String(), updatedObject, tc.objectName[i]); err != nil {
					t.Fatalf("error getting updated statefulset: %v", err)
				}
				assert.Equal(t, tc.expectedReplicas, *updatedObject.Spec.Replicas)
			}
		})
	}
}
