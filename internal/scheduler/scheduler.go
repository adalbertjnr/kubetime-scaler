package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	downscalergov1alpha1 "github.com/adalbertjnr/downscaler-operator/api/v1alpha1"
	"github.com/adalbertjnr/downscaler-operator/internal/client"
	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	downscaleReplicas int = 0
	upscaleReplicas   int = 1
)

type Downscaler struct {
	app downscalergov1alpha1.Downscaler

	log logr.Logger

	client *client.APIClient

	cron *cron.Cron

	cancelFunc context.CancelFunc
}

func (dc *Downscaler) Client(c *client.APIClient) *Downscaler {
	dc.client = c
	return dc
}

func (dc *Downscaler) Run() (ctrl.Result, error) {
	if err := dc.initializeCronClient(); err != nil {
		return ctrl.Result{}, err
	}

	dc.initializeCronTasks()

	return ctrl.Result{}, nil
}

func (dc *Downscaler) addCronJob(scaleStr, namespace string, replicas int) {
	expression := buildCronExpression(dc.recurrence(), scaleStr)
	_, err := dc.cron.AddFunc(expression, dc.job(namespace, replicas))
	if err != nil {
		slog.Error("cron", "scheduling error", err)
	}
}

func (dc *Downscaler) initializeCronTasks() {
	dc.cleanCronEntries()

	for _, rule := range dc.rules() {
		for _, namespace := range rule.Namespaces {
			if namespace.Ignored(excluded(dc.excludedNamespaces(), func(namespace string) (string, struct{}) { return namespace, struct{}{} })) {
				dc.log.Info("cron", "ignoring namespace during cron task initialiation", namespace.String())
				continue
			}
			dc.addCronJob(rule.UpscaleTime, namespace.String(), upscaleReplicas)
			dc.addCronJob(rule.DownscaleTime, namespace.String(), downscaleReplicas)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	dc.cancelFunc = cancel

	go dc.notifyCronEntries(ctx)

	dc.cron.Start()
}

func (dc *Downscaler) notifyCronEntries(ctx context.Context) {
	interval := dc.app.Spec.Config.CronLoggerInterval
	if interval <= 0 {
		interval = 300
	}

	ticker := time.NewTicker(time.Second * time.Duration(interval))
	dc.log.Info("cron notification started", "interval", interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, entry := range dc.cron.Entries() {
				dc.log.Info("cron", "entryID", entry.ID, "nextRun", entry.Next)
			}
		}
	}
}

func (dc *Downscaler) job(namespace string, replicas int) func() {
	return func() {
		deployments, err := dc.client.GetDeployments(namespace)
		if err != nil {
			slog.Error("client", "error get deployments", err)
			return
		}

		for _, deployment := range deployments.Items {
			before := *deployment.Spec.Replicas

			if err := dc.client.PatchDeployment(replicas, &deployment); err != nil {
				slog.Error("client", "error patching deployment", err)
				return
			}

			dc.log.Info("client patching deployment replicas", "before", before, "after", replicas)
		}
	}
}

func (s *Downscaler) Add(ctx context.Context, app downscalergov1alpha1.Downscaler) *Downscaler {
	s.app = app
	return s
}

func (s *Downscaler) Logger(log logr.Logger) *Downscaler {
	s.log = log
	return s
}

func (dc *Downscaler) initializeCronClient() error {
	if dc.cron == nil {
		downscaler, err := dc.client.GetDownscaler()
		if err != nil {
			return err
		}

		location, err := time.LoadLocation(downscaler.Spec.Schedule.TimeZone)
		if err != nil {
			return err
		}

		cron := cron.New(cron.WithLocation(location))
		dc.cron = cron
	}

	return nil
}

func buildCronExpression(recurrence, timeStr string) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Error("Invalid time format", "timeStr", timeStr, "error", err)
		return "0 0 * * *"
	}

	return fmt.Sprintf("%d %d * * %s", t.Minute(), t.Hour(), recurrence)
}

func (dc *Downscaler) cleanCronEntries() {
	entries := dc.cron.Entries()

	if dc.cancelFunc != nil {
		dc.cancelFunc()
	}

	if len(entries) > 0 {
		for _, entry := range entries {
			dc.cron.Remove(entry.ID)
			dc.log.Info("cron", "cleaning entryID", entry.ID)
		}
	}
}

func (dc *Downscaler) rules() []downscalergov1alpha1.Rules {
	return dc.app.Spec.NamespacesRules.Include.WithRulesByNamespaces.Rules
}

func (dc *Downscaler) excludedNamespaces() []string {
	return dc.app.Spec.NamespacesRules.Exclude.Namespaces
}

func (dc *Downscaler) recurrence() string {
	return dc.app.Spec.Schedule.Recurrence
}

func excluded(collection []string, fn func(namespace string) (string, struct{})) map[string]struct{} {
	result := make(map[string]struct{}, len(collection))
	for i := range collection {
		namespace, empty := fn(collection[i])
		result[namespace] = empty
	}
	return result
}
