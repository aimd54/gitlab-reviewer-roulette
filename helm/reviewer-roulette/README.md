# GitLab Reviewer Roulette Helm Chart

This Helm chart deploys the GitLab Reviewer Roulette Bot to a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PostgreSQL database (managed service or self-hosted)
- Redis cache (managed service or self-hosted)
- GitLab 16.11+ (self-hosted or GitLab.com)

## Installing the Chart

### From Local Directory

```bash
# Clone the repository
git clone https://github.com/aimd54/gitlab-reviewer-roulette.git
cd gitlab-reviewer-roulette/helm/reviewer-roulette

# Install
helm install my-reviewer-roulette . -f values.yaml
```

### From Packaged Chart

```bash
# Package the chart
helm package .

# Install from package
helm install my-reviewer-roulette reviewer-roulette-1.0.0.tgz
```

## Configuration

### Required Values

You **must** provide these values for the chart to work:

```yaml
config:
  gitlab:
    url: "https://your-gitlab.com"
    token: "YOUR_GITLAB_TOKEN"  # Personal access token with 'api' scope
    webhookSecret: "YOUR_WEBHOOK_SECRET"  # Secret for webhook validation

  postgres:
    host: "your-postgresql-host"
    password: "YOUR_DB_PASSWORD"

  redis:
    host: "your-redis-host"
```

### Recommended Production Values

Create a `values-production.yaml` file:

```yaml
# values-production.yaml

replicaCount: 2  # High availability

image:
  repository: registry.example.com/reviewer-roulette
  tag: "1.5.0"
  pullPolicy: IfNotPresent

# Resource limits for production
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

# Enable autoscaling
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 70

# Enable PodDisruptionBudget
podDisruptionBudget:
  enabled: true
  minAvailable: 1

# Ingress configuration
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: reviewer-roulette.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: reviewer-roulette-tls
      hosts:
        - reviewer-roulette.example.com

# ServiceMonitor for Prometheus Operator
metrics:
  serviceMonitor:
    enabled: true
    interval: 30s

# Production configuration
config:
  server:
    environment: production
    language: en

  logging:
    level: info
    format: json

  # Your GitLab, PostgreSQL, Redis config here
  # ...

  # Customize teams
  teams:
    - name: team-frontend
      members:
        - username: alice
          role: dev
        # ... more members

  # Adjust badge thresholds for your team size
  badges:
    - name: "speed_demon"
      displayName: "Speed Demon"
      description: "⚡ Reviews in less than 2 hours"
      icon: "⚡"
      tier: "bronze"
      criteria:
        metric: avg_ttfr
        operator: "<"
        value: 120  # Adjust this
        period: month
```

Install with production values:

```bash
helm install my-reviewer-roulette . \
  -f values.yaml \
  -f values-production.yaml \
  --create-namespace \
  --namespace reviewer-roulette
```

### Using Secrets

For sensitive values, use `--set` or external secrets:

```bash
# Using --set
helm install my-reviewer-roulette . \
  --set config.gitlab.token=$GITLAB_TOKEN \
  --set config.gitlab.webhookSecret=$WEBHOOK_SECRET \
  --set config.postgres.password=$DB_PASSWORD

# Or use an external secret (recommended)
# See: https://external-secrets.io/
```

## Upgrading the Chart

```bash
# Upgrade to new version
helm upgrade my-reviewer-roulette . \
  -f values-production.yaml \
  --namespace reviewer-roulette

# Upgrade with new image
helm upgrade my-reviewer-roulette . \
  --set image.tag=1.6.0 \
  --namespace reviewer-roulette
```

## Uninstalling the Chart

```bash
helm uninstall my-reviewer-roulette --namespace reviewer-roulette
```

**Note:** This will not delete the PostgreSQL database or Redis data if using external services.

## Post-Installation Steps

After installing the chart:

### 1. Verify Deployment

```bash
# Check pod status
kubectl get pods -n reviewer-roulette

# Check logs
kubectl logs -n reviewer-roulette -l app.kubernetes.io/name=reviewer-roulette -f

# Check health
kubectl port-forward -n reviewer-roulette svc/my-reviewer-roulette 8080:8080
curl http://localhost:8080/health
```

### 2. Initialize Users

Run the init command to sync users from GitLab:

```bash
kubectl exec -n reviewer-roulette -it deployment/my-reviewer-roulette -- \
  /app/init --config /app/config/config.yaml
```

### 3. Configure GitLab Webhook

In your GitLab project:

1. Go to **Settings** > **Webhooks**
2. Add a new webhook:
   - **URL**: `https://reviewer-roulette.example.com/webhook/gitlab`
   - **Secret Token**: Your webhook secret
   - **Trigger**: Merge Request events, Note events
   - **SSL verification**: Enable (recommended)
3. Test the webhook

### 4. Test the /roulette Command

1. Create a test merge request
2. Add a comment: `/roulette`
3. The bot should respond with 3 reviewer assignments

## Configuration Reference

### Common Values

| Parameter                 | Description             | Default                    |
| ------------------------- | ----------------------- | -------------------------- |
| `replicaCount`            | Number of replicas      | `2`                        |
| `image.repository`        | Image repository        | `gitlab-reviewer-roulette` |
| `image.tag`               | Image tag               | `Chart.appVersion`         |
| `image.pullPolicy`        | Image pull policy       | `IfNotPresent`             |
| `service.type`            | Kubernetes service type | `ClusterIP`                |
| `service.port`            | Service port            | `8080`                     |
| `ingress.enabled`         | Enable ingress          | `false`                    |
| `resources.limits.cpu`    | CPU limit               | `1000m`                    |
| `resources.limits.memory` | Memory limit            | `1Gi`                      |
| `autoscaling.enabled`     | Enable HPA              | `false`                    |
| `autoscaling.minReplicas` | Minimum replicas        | `2`                        |
| `autoscaling.maxReplicas` | Maximum replicas        | `5`                        |

### Application Configuration

| Parameter                     | Description                          | Default             |
| ----------------------------- | ------------------------------------ | ------------------- |
| `config.server.port`          | Application port                     | `8080`              |
| `config.server.environment`   | Environment (development/production) | `production`        |
| `config.server.language`      | Bot language (en/fr)                 | `en`                |
| `config.gitlab.url`           | GitLab instance URL                  | `` (required)       |
| `config.gitlab.token`         | GitLab API token                     | `` (required)       |
| `config.gitlab.webhookSecret` | Webhook secret                       | `` (required)       |
| `config.postgres.host`        | PostgreSQL host                      | `postgresql`        |
| `config.postgres.port`        | PostgreSQL port                      | `5432`              |
| `config.postgres.database`    | Database name                        | `reviewer_roulette` |
| `config.redis.host`           | Redis host                           | `redis-master`      |
| `config.redis.port`           | Redis port                           | `6379`              |
| `scheduler.enabled`           | Enable daily scheduler               | `true`              |
| `scheduler.timezone`          | Scheduler timezone                   | `UTC`               |

See `values.yaml` for all available parameters.

## Monitoring

### Prometheus Metrics

Metrics are exposed on port `9090` at `/metrics`:

```bash
# Port-forward metrics
kubectl port-forward -n reviewer-roulette svc/my-reviewer-roulette-metrics 9090:9090

# View metrics
curl http://localhost:9090/metrics
```

### ServiceMonitor

If using Prometheus Operator, enable ServiceMonitor:

```yaml
metrics:
  serviceMonitor:
    enabled: true
    interval: 30s
```

### Grafana Dashboards

Import the provided Grafana dashboards from the main repository:

- Team Performance Dashboard
- Reviewer Statistics Dashboard
- Review Quality Dashboard

## Troubleshooting

### Pods Not Starting

```bash
# Check pod events
kubectl describe pod -n reviewer-roulette <pod-name>

# Check logs
kubectl logs -n reviewer-roulette <pod-name>
kubectl logs -n reviewer-roulette <pod-name> -c migrate  # Init container
```

### Database Connection Issues

```bash
# Test database connection
kubectl exec -n reviewer-roulette -it deployment/my-reviewer-roulette -- \
  psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB
```

### Webhook Not Working

1. Check ingress is properly configured
2. Verify webhook secret matches
3. Check GitLab can reach the webhook URL
4. Review application logs for webhook errors

### Common Issues

| Issue                                              | Solution                                                  |
| -------------------------------------------------- | --------------------------------------------------------- |
| `Error: failed to download "reviewer-roulette"`    | Ensure Helm repository is added and updated               |
| `Error: INSTALLATION FAILED: cannot re-use a name` | Use `helm upgrade` or `helm uninstall` first              |
| `Error: Kubernetes cluster unreachable`            | Check `kubectl` context: `kubectl config current-context` |
| `CrashLoopBackOff` on migration                    | Check database credentials and connectivity               |
| `ImagePullBackOff`                                 | Verify image repository and credentials                   |

## Development

### Linting the Chart

```bash
helm lint .
```

### Testing Templates

```bash
# Render templates locally
helm template my-reviewer-roulette . -f values.yaml

# Debug specific template
helm template my-reviewer-roulette . --debug --show-only templates/deployment.yaml
```

### Validating Against Cluster

```bash
# Dry run installation
helm install my-reviewer-roulette . --dry-run --debug
```

## Support

- Documentation: <https://github.com/aimd54/gitlab-reviewer-roulette>
- Issues: <https://github.com/aimd54/gitlab-reviewer-roulette/issues>
- Full Docs: See [README](../../README.md)

## License

MIT License - See `../../LICENSE`
