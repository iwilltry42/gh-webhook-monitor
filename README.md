# gh-webhook-monitor

Prometheus Metrics Exporter for GitHub Repository Webhook Statuses

## Overview

- Exposes metrics on `:8080/metrics`

## Configuration

Via Environment Variables:

| Variable Name         | Value Type        | Description                                                                       |
|-----------------------|-------------------|-----------------------------------------------------------------------------------|
| `GWM_GH_APP_ID`       | int               | ID of your GitHub App                                                             |
| `GWM_GH_APP_PEM`      | string            | Path to the private key PEM file of your GitHub App                               |
| `GWM_GH_APP_INST_ID`  | int               | ID of the Installation of your GitHub App                                         |
| `GWM_REPOS`           | string            | Comma-separated list of repositories to check the webhooks for                    |
| `GWM_WAIT_TIME`       | time.Duration     | Time to wait between each loop (important for request limits on the GitHub API)   |
| `GWM_WEBHOOK_REGEXP`  | string (regexp)   | Regular Expression to filter for specific webhook target URLs (e.g. `.*jenkins.*`)|