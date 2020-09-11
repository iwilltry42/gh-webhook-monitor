package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	webhookLastStatusCode = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gh_webhook_last_status_code",
		Help: "The last HTTP status code per webhook",
	}, []string{
		"repository",
		"webhook",
		"code",
	})

	repositoryFailedWebhookList = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gh_webhooks_repository_list_failed_total",
		Help: "Total number of failed webhook lists per repository",
	}, []string{
		"repository",
		"error",
	})
)
