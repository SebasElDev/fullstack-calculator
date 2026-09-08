#!/usr/bin/env bash
#
# bootstrap.sh — first-time AWS provisioning for fullstack-calculator.
#
# Deploys the foundation stack (ECR repo, App Runner access role, optional
# GitHub OIDC provider + deploy role), then calls deploy.sh to build, push
# and stand up the App Runner service. Finally prints the service URL and
# the `gh variable set` commands needed to wire up CI/CD.
#
# Usage: ./deploy/aws/bootstrap.sh
#
# Configurable via environment variables (defaults shown):
#   AWS_REGION=us-east-1
#   PROJECT_NAME=fullstack-calculator
#   FOUNDATION_STACK=${PROJECT_NAME}-foundation
#   SERVICE_STACK=${PROJECT_NAME}-service
#   GITHUB_REPOSITORY=SebasElDev/fullstack-calculator
#   GITHUB_BRANCH=main
set -euo pipefail

AWS_REGION="${AWS_REGION:-us-east-1}"
PROJECT_NAME="${PROJECT_NAME:-fullstack-calculator}"
FOUNDATION_STACK="${FOUNDATION_STACK:-${PROJECT_NAME}-foundation}"
SERVICE_STACK="${SERVICE_STACK:-${PROJECT_NAME}-service}"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-SebasElDev/fullstack-calculator}"
GITHUB_BRANCH="${GITHUB_BRANCH:-main}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Checking for an existing GitHub OIDC provider in this account..."
CREATE_OIDC_PROVIDER="true"
if aws iam list-open-id-connect-providers --region "$AWS_REGION" \
    | grep -q "token.actions.githubusercontent.com"; then
  echo "    Found an existing token.actions.githubusercontent.com provider; reusing it."
  CREATE_OIDC_PROVIDER="false"
else
  echo "    None found; the foundation stack will create one."
fi

echo "==> Deploying foundation stack: ${FOUNDATION_STACK}"
aws cloudformation deploy \
  --region "$AWS_REGION" \
  --template-file "${SCRIPT_DIR}/foundation.yaml" \
  --stack-name "$FOUNDATION_STACK" \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameter-overrides \
    ProjectName="$PROJECT_NAME" \
    GitHubRepository="$GITHUB_REPOSITORY" \
    GitHubBranch="$GITHUB_BRANCH" \
    CreateGitHubOidcProvider="$CREATE_OIDC_PROVIDER" \
  --no-fail-on-empty-changeset

echo "==> Reading foundation stack outputs"
ECR_REPOSITORY_URI="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
  --stack-name "$FOUNDATION_STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='EcrRepositoryUri'].OutputValue" --output text)"
ECR_REPOSITORY_NAME="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
  --stack-name "$FOUNDATION_STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='EcrRepositoryName'].OutputValue" --output text)"
APP_RUNNER_ACCESS_ROLE_ARN="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
  --stack-name "$FOUNDATION_STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='AppRunnerEcrAccessRoleArn'].OutputValue" --output text)"
GITHUB_DEPLOY_ROLE_ARN="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
  --stack-name "$FOUNDATION_STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='GitHubDeployRoleArn'].OutputValue" --output text)"

echo "==> Building, pushing and deploying the service"
AWS_REGION="$AWS_REGION" \
PROJECT_NAME="$PROJECT_NAME" \
SERVICE_STACK="$SERVICE_STACK" \
ECR_REPOSITORY_URI="$ECR_REPOSITORY_URI" \
APP_RUNNER_ACCESS_ROLE_ARN="$APP_RUNNER_ACCESS_ROLE_ARN" \
  "${SCRIPT_DIR}/deploy.sh"

SERVICE_URL="$(aws cloudformation describe-stacks --region "$AWS_REGION" \
  --stack-name "$SERVICE_STACK" \
  --query "Stacks[0].Outputs[?OutputKey=='ServiceUrl'].OutputValue" --output text)"

cat <<SUMMARY

==================================================================
Bootstrap complete.

  Service URL: ${SERVICE_URL}

Configure these GitHub Actions repository variables for CI/CD
(requires the gh CLI, run from the repository root):

  gh variable set AWS_ROLE_ARN --body "${GITHUB_DEPLOY_ROLE_ARN}"
  gh variable set AWS_REGION --body "${AWS_REGION}"
  gh variable set ECR_REPOSITORY --body "${ECR_REPOSITORY_NAME}"
  gh variable set SERVICE_STACK_NAME --body "${SERVICE_STACK}"
  gh variable set APP_RUNNER_ACCESS_ROLE_ARN --body "${APP_RUNNER_ACCESS_ROLE_ARN}"
==================================================================
SUMMARY
