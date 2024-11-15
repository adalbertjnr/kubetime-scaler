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

func (d *Downscaler) Client(c *client.APIClient) *Downscaler {
	d.client = c
	return d
}

func (d *Downscaler) Run() (ctrl.Result, error) {
	d.initializeCronClient()

	d.initializeCronTasks()

	return ctrl.Result{}, nil
}

func (d *Downscaler) initializeCronTasks() {
	d.clean()

	spec := d.app.Spec

	for _, rule := range spec.NamespacesRules.Include.WithRulesByNamespaces.Rules {
		for _, namespace := range rule.Namespaces {

			upscaleExpression := buildCronExpression(spec.Schedule.Recurrence, rule.UpscaleTime)
			_, err := d.cron.AddFunc(upscaleExpression, d.job(namespace, upscaleReplicas))
			if err != nil {
				slog.Error("cron", "scheduling error", err)
				continue
			}

			downscaleExpression := buildCronExpression(spec.Schedule.Recurrence, rule.DownscaleTime)
			_, err = d.cron.AddFunc(downscaleExpression, d.job(namespace, downscaleReplicas))
			if err != nil {
				slog.Error("cron", "scheduling error", err)
				continue
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelFunc = cancel

	go d.notifyCronEntries(ctx)

	d.cron.Start()
}

func (d *Downscaler) notifyCronEntries(ctx context.Context) {
	interval := d.app.Spec.Config.CronLoggerInterval
	if interval <= 0 {
		interval = 300
	}

	ticker := time.NewTicker(time.Second * time.Duration(interval))
	d.log.Info("cron notification started", "interval", interval)
	// slog.Info("cron notification started", "interval", interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, entry := range d.cron.Entries() {
				d.log.Info("cron", "entryID", entry.ID, "nextRun", entry.Next)
				// slog.Info("cron", "entryID", entry.ID, "nextRun", entry.Next)
			}
		}
	}
}

func (d *Downscaler) job(namespace string, replicas int) func() {
	return func() {
		deployments, err := d.client.GetDeployments(namespace)
		if err != nil {
			slog.Error("client", "error get deployments", err)
			return
		}

		for _, deployment := range deployments.Items {
			before := *deployment.Spec.Replicas

			if err := d.client.PatchDeployment(replicas, &deployment); err != nil {
				slog.Error("client", "error patching deployment", err)
				return
			}

			d.log.Info("client patching deployment replicas", "before", before, "after", replicas)
			// slog.Info("client patching deployment replicas", "before", before, "after", replicas)
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

func (d *Downscaler) initializeCronClient() (ctrl.Result, error) {
	if d.cron == nil {
		downscaler, err := d.client.GetDownscaler()
		if err != nil {
			return ctrl.Result{}, err
		}

		location, err := time.LoadLocation(downscaler.Spec.Schedule.TimeZone)
		if err != nil {
			return ctrl.Result{}, err
		}

		cron := cron.New(cron.WithLocation(location))
		d.cron = cron
	}

	return ctrl.Result{}, nil
}

func buildCronExpression(recurrence, timeStr string) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Error("Invalid time format", "timeStr", timeStr, "error", err)
		return "0 0 * * *"
	}

	return fmt.Sprintf("%d %d * * %s", t.Minute(), t.Hour(), recurrence)
}

func (d *Downscaler) clean() {
	entries := d.cron.Entries()

	if d.cancelFunc != nil {
		d.cancelFunc()
	}

	if len(entries) > 0 {
		for _, entry := range entries {
			d.cron.Remove(entry.ID)
			d.log.Info("cron", "cleaning entryID", entry.ID)
			// slog.Info("cron", "cleaning entryID", entry.ID)
		}

	}
}
