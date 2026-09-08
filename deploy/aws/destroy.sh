#!/usr/bin/env bash
#
# destroy.sh — tear down all AWS resources for fullstack-calculator: the
# App Runner service stack, all images in the ECR repository, and the
# foundation stack.
#
# Usage: ./deploy/aws/destroy.sh
#   FORCE=1 ./deploy/aws/destroy.sh   # skip the confirmation prompt
#
# Configurable via environment variables (defaults shown):
#   AWS_REGION=us-east-1
#   PROJECT_NAME=fullstack-calculator
#   FOUNDATION_STACK=${PROJECT_NAME}-foundation
#   SERVICE_STACK=${PROJECT_NAME}-service
set -euo pipefail

AWS_REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="${PROJECT_NAME:-fullstack-calculator}"
FOUNDATION_STACK="${FOUNDATION_STACK:-${PROJECT_NAME}-foundation}"
SERVICE_STACK="${SERVICE_STACK:-${PROJECT_NAME}-service}"

if [[ "${FORCE:-0}" != "1" ]]; then
  read -r -p "This will permanently delete stacks '${SERVICE_STACK}' and '${FOUNDATION_STACK}' and all images in their ECR repository, in region ${AWS_REGION}. Type 'yes' to continue: " confirm
  if [[ "$confirm" != "yes" ]]; then
    echo "Aborted."
    exit 1
  fi
fi

echo "==> Deleting service stack: ${SERVICE_STACK}"
if aws cloudformation describe-stacks --region "$AWS_REGION" --stack-name "$SERVICE_STACK" >/dev/null 2>&1; then
  aws cloudformation delete-stack --region "$AWS_REGION" --stack-name "$SERVICE_STACK"
  aws cloudformation wait stack-delete-complete --region "$AWS_REGION" --stack-name "$SERVICE_STACK"
else
  echo "    Stack not found; skipping."
fi

echo "==> Emptying ECR repository: ${PROJECT_NAME}"
if aws ecr describe-repositories --region "$AWS_REGION" --repository-names "$PROJECT_NAME" >/dev/null 2>&1; then
  IMAGE_IDS="$(aws ecr list-images --region "$AWS_REGION" --repository-name "$PROJECT_NAME" \
    --query 'imageIds[*]' --output json)"
  if [[ "$IMAGE_IDS" != "[]" ]]; then
    aws ecr batch-delete-image --region "$AWS_REGION" --repository-name "$PROJECT_NAME" \
      --image-ids "$IMAGE_IDS" >/dev/null
  else
    echo "    Repository already empty."
  fi
else
  echo "    Repository not found; skipping."
fi

echo "==> Deleting foundation stack: ${FOUNDATION_STACK}"
if aws cloudformation describe-stacks --region "$AWS_REGION" --stack-name "$FOUNDATION_STACK" >/dev/null 2>&1; then
  aws cloudformation delete-stack --region "$AWS_REGION" --stack-name "$FOUNDATION_STACK"
  aws cloudformation wait stack-delete-complete --region "$AWS_REGION" --stack-name "$FOUNDATION_STACK"
else
  echo "    Stack not found; skipping."
fi

echo "==> Done."
