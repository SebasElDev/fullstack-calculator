#!/usr/bin/env bash
#
# deploy.sh — build the production image, push it to ECR, deploy the App
# Runner service stack and wait for it to become healthy.
#
# Usage: ./deploy/aws/deploy.sh
#
# Configurable via environment variables (defaults shown):
#   AWS_REGION=us-east-1
#   PROJECT_NAME=fullstack-calculator
#   FOUNDATION_STACK=${PROJECT_NAME}-foundation
#   SERVICE_STACK=${PROJECT_NAME}-service
#   ECR_REPOSITORY_URI            (read from the foundation stack if unset)
#   APP_RUNNER_ACCESS_ROLE_ARN    (read from the foundation stack if unset)
set -euo pipefail

AWS_REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="${PROJECT_NAME:-fullstack-calculator}"
FOUNDATION_STACK="${FOUNDATION_STACK:-${PROJECT_NAME}-foundation}"
SERVICE_STACK="${SERVICE_STACK:-${PROJECT_NAME}-service}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

if [[ -z "${ECR_REPOSITORY_URI:-}" ]]; then
  ECR_REPOSITORY_URI="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
    --stack-name "$FOUNDATION_STACK" \
    --query "Stacks[0].Outputs[?OutputKey=='EcrRepositoryUri'].OutputValue" --output text)"
fi

if [[ -z "${APP_RUNNER_ACCESS_ROLE_ARN:-}" ]]; then
  APP_RUNNER_ACCESS_ROLE_ARN="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
    --stack-name "$FOUNDATION_STACK" \
    --query "Stacks[0].Outputs[?OutputKey=='AppRunnerEcrAccessRoleArn'].OutputValue" --output text)"
fi

VERSION="$(git -C "$REPO_ROOT" rev-parse --short HEAD)"
IMAGE_URI="${ECR_REPOSITORY_URI}:${VERSION}"

echo "==> Logging in to ECR"
aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "${ECR_REPOSITORY_URI%%/*}"

echo "==> Building and pushing ${IMAGE_URI}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg VERSION="$VERSION" \
  --push \
  -t "$IMAGE_URI" \
  -f "${REPO_ROOT}/Dockerfile" \
  "$REPO_ROOT"

echo "==> Deploying service stack: ${SERVICE_STACK}"
aws cloudformation deploy \
  --region "$AWS_REGION" \
  --template-file "${SCRIPT_DIR}/service.yaml" \
  --stack-name "$SERVICE_STACK" \
  --parameter-overrides \
    ProjectName="$PROJECT_NAME" \
    ImageUri="$IMAGE_URI" \
    AccessRoleArn="$APP_RUNNER_ACCESS_ROLE_ARN" \
  --capabilities CAPABILITY_NAMED_IAM \
  --no-fail-on-empty-changeset

SERVICE_ARN="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
  --stack-name "$SERVICE_STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='ServiceArn'].OutputValue" --output text)"

echo "==> Waiting for App Runner service to reach RUNNING"
elapsed=0
timeout=900
interval=15
while true; do
  status="$(aws apprunner describe-service --region "$AWS_REGION" \
    --service-arn "$SERVICE_ARN" --query "Service.Status" --output text)"
  echo "    status=${status} (elapsed ${elapsed}s)"

  if [[ "$status" == "RUNNING" ]]; then
    break
  fi
  if [[ "$status" == *_FAILED ]]; then
    echo "App Runner service entered a failed state: ${status}" >&2
    exit 1
  fi
  if (( elapsed >= timeout )); then
    echo "Timed out waiting for App Runner service to reach RUNNING" >&2
    exit 1
  fi
  sleep "$interval"
  elapsed=$((elapsed + interval))
done

SERVICE_URL="$(aws apprunner describe-service --region "$AWS_REGION" \
  --service-arn "$SERVICE_ARN" --query "Service.ServiceUrl" --output text)"

echo "==> Checking health endpoint"
curl -fsS "https://${SERVICE_URL}/health"
echo

echo "==> Deployed: https://${SERVICE_URL}"
