---
title: Developer Workflows
description: Use secretctl in your daily development workflow for local development, CI/CD, and team collaboration.
sidebar_position: 2
---

# Developer Workflows

This guide covers common developer scenarios where secretctl streamlines secrets management, from local development to production deployments.

## Local Development

### Replace Scattered .env Files

**Problem:** `.env` files scattered across projects, easy to commit accidentally, hard to keep in sync.

**Solution:** Centralize secrets in secretctl vault.

```bash
# Instead of managing multiple .env files
# Store secrets once in the vault
echo "sk-abc123" | secretctl set OPENAI_API_KEY
echo "postgres://user:pass@localhost/db" | secretctl set DATABASE_URL

# Run your app with injected secrets
secretctl run -k OPENAI_API_KEY -k DATABASE_URL -- npm start
```

### Project-Specific Secrets

Organize secrets by project using hierarchical keys:

```bash
# Project A secrets
echo "key-a" | secretctl set projectA/api_key
echo "db-a" | secretctl set projectA/database_url

# Project B secrets
echo "key-b" | secretctl set projectB/api_key
echo "db-b" | secretctl set projectB/database_url

# Run with project-specific secrets
secretctl run -k "projectA/*" -- ./run-project-a.sh
secretctl run -k "projectB/*" -- ./run-project-b.sh
```

### Multiple Environments

Use environment aliases for seamless environment switching:

```yaml
# ~/.secretctl/mcp-policy.yaml
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

```bash
# Same command, different environments
secretctl run --env=dev -k "db/*" -- ./app
secretctl run --env=prod -k "db/*" -- ./app
```

## CI/CD Integration

### GitHub Actions

Inject secrets into GitHub Actions workflows:

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install secretctl
        run: |
          curl -fsSL https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64 -o secretctl
          chmod +x secretctl
          sudo mv secretctl /usr/local/bin/

      - name: Initialize vault
        env:
          SECRETCTL_PASSWORD: ${{ secrets.SECRETCTL_PASSWORD }}
        run: |
          # Restore vault from secure storage or initialize
          secretctl init

      - name: Deploy with secrets
        env:
          SECRETCTL_PASSWORD: ${{ secrets.SECRETCTL_PASSWORD }}
        run: |
          secretctl run -k DEPLOY_TOKEN -k AWS_ACCESS_KEY -- ./deploy.sh
```

### GitLab CI

```yaml
# .gitlab-ci.yml
deploy:
  stage: deploy
  script:
    - curl -fsSL https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64 -o secretctl
    - chmod +x secretctl && mv secretctl /usr/local/bin/
    - secretctl run -k "deploy/*" -- ./deploy.sh
  variables:
    SECRETCTL_PASSWORD: $SECRETCTL_PASSWORD
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any
    environment {
        SECRETCTL_PASSWORD = credentials('secretctl-password')
    }
    stages {
        stage('Deploy') {
            steps {
                sh '''
                    secretctl run -k DEPLOY_TOKEN -- ./deploy.sh
                '''
            }
        }
    }
}
```

## Docker Workflows

### Development with Docker Compose

Generate `.env` for Docker Compose:

```bash
# Export secrets for Docker Compose
secretctl export -k "docker/*" -o .env

# Or run docker-compose directly
secretctl run -k "docker/*" -- docker-compose up
```

```yaml
# docker-compose.yml
services:
  app:
    build: .
    env_file:
      - .env
```

### Docker Build Arguments

Pass secrets as build arguments securely:

```bash
# Don't embed secrets in Dockerfile
# Instead, pass at build time
secretctl run -k GITHUB_TOKEN -- docker build \
  --build-arg GITHUB_TOKEN=$GITHUB_TOKEN \
  -t myapp .
```

### Docker Run

Inject secrets at runtime:

```bash
secretctl run -k "app/*" -- docker run \
  -e API_KEY=$API_KEY \
  -e DATABASE_URL=$DATABASE_URL \
  myapp
```

## Database Connections

### PostgreSQL

```bash
# Store database credentials
echo "prod-db.example.com" | secretctl set db/host
echo "myuser" | secretctl set db/user
echo "secret123" | secretctl set db/password

# Connect using psql
secretctl run -k "db/*" -- psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST/mydb"
```

### MySQL

```bash
secretctl run -k "mysql/*" -- mysql \
  -h $MYSQL_HOST \
  -u $MYSQL_USER \
  -p$MYSQL_PASSWORD \
  mydb
```

### Database Migrations

```bash
# Run migrations with database credentials
secretctl run -k DATABASE_URL -- npx prisma migrate deploy
secretctl run -k DATABASE_URL -- rails db:migrate
secretctl run -k "db/*" -- flyway migrate
```

## Cloud Provider CLIs

### AWS CLI

```bash
# Store AWS credentials
echo "AKIAIOSFODNN7EXAMPLE" | secretctl set aws/access_key_id
echo "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" | secretctl set aws/secret_access_key

# Run AWS commands
secretctl run -k "aws/*" -- aws s3 ls
secretctl run -k "aws/*" -- aws ec2 describe-instances
```

### Google Cloud

```bash
# Store service account key path or credentials
echo "/path/to/service-account.json" | secretctl set gcp/credentials_file

# Run gcloud commands
secretctl run -k "gcp/*" -- gcloud compute instances list
```

### Azure CLI

```bash
# Store Azure credentials
echo "your-client-id" | secretctl set azure/client_id
echo "your-client-secret" | secretctl set azure/client_secret

secretctl run -k "azure/*" -- az vm list
```

## API Development

### Testing APIs with curl

```bash
# Store API keys
echo "sk-live-xxx" | secretctl set stripe/api_key
echo "Bearer xxx" | secretctl set github/token

# Make authenticated requests
secretctl run -k "stripe/*" -- curl -u $STRIPE_API_KEY: https://api.stripe.com/v1/charges

secretctl run -k "github/*" -- curl -H "Authorization: $GITHUB_TOKEN" https://api.github.com/user
```

### Postman/Insomnia

Export secrets for API testing tools:

```bash
# Export as JSON for import
secretctl export -k "api/*" --format=json -o api-secrets.json
```

## Kubernetes Workflows

### Create Kubernetes Secrets

```bash
# Export and create k8s secret
secretctl export -k "app/*" --format=json | \
  kubectl create secret generic app-secrets --from-env-file=/dev/stdin

# Or using individual keys
secretctl run -k "k8s/*" -- kubectl create secret generic db-creds \
  --from-literal=username=$DB_USER \
  --from-literal=password=$DB_PASSWORD
```

### Helm Deployments

```bash
# Pass secrets to Helm
secretctl run -k "helm/*" -- helm upgrade myapp ./chart \
  --set database.password=$DB_PASSWORD \
  --set api.key=$API_KEY
```

## Terraform Workflows

### Environment Variables

```bash
# Terraform reads TF_VAR_* environment variables
secretctl run -k "terraform/*" -- terraform apply

# Secrets stored as:
# terraform/TF_VAR_db_password -> TF_VAR_DB_PASSWORD
```

### Backend Configuration

```bash
# Configure remote state backend
secretctl run -k AWS_ACCESS_KEY -k AWS_SECRET_KEY -- terraform init
```

## Best Practices

### Naming Conventions

Organize secrets consistently:

```
# By service
aws/access_key
aws/secret_key
stripe/api_key
stripe/webhook_secret

# By environment
dev/database_url
staging/database_url
prod/database_url

# By project
projectA/api_key
projectB/api_key
```

### Rotation Workflow

```bash
# Generate new password
NEW_PASS=$(secretctl generate -l 32)

# Update secret
echo "$NEW_PASS" | secretctl set db/password \
  --notes="Rotated on $(date +%Y-%m-%d)"

# Restart services
secretctl run -k "db/*" -- ./restart-services.sh
```

### Team Onboarding

```bash
# Document required secrets for new team members
secretctl list > required-secrets.txt

# New member creates their vault and adds secrets
secretctl init
echo "my-api-key" | secretctl set API_KEY
```

### Audit Trail

```bash
# Review access patterns
secretctl audit export --format=json -o audit-$(date +%Y%m%d).json

# Check for unusual access
secretctl audit export | jq '.[] | select(.source == "mcp")'
```

## Troubleshooting

### Secret Not Found

```bash
# Verify secret exists
secretctl list | grep API_KEY

# Check exact key name (case-sensitive)
secretctl get API_KEY
```

### Environment Variable Not Set

```bash
# Debug: print injected variables
secretctl run -k "api/*" -- env | grep API

# Check wildcard pattern matches
secretctl list | grep "api/"
```

### CI/CD Password Issues

```bash
# Ensure SECRETCTL_PASSWORD is set
if [ -z "$SECRETCTL_PASSWORD" ]; then
  echo "Error: SECRETCTL_PASSWORD not set"
  exit 1
fi
```

## Next Steps

- [AI Agent Integration](/docs/use-cases/ai-agent-integration) - MCP and Claude Code
- [Running Commands](/docs/guides/cli/running-commands) - Detailed run command guide
- [Exporting Secrets](/docs/guides/cli/exporting-secrets) - Export formats and options
