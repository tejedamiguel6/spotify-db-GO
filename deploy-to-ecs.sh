#!/bin/bash

# Spotify Track DB - AWS ECS Deployment Script
# This script automates the deployment process to AWS ECS

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Spotify Track DB - AWS ECS Deployment${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Configuration
REGION="us-east-2"
REPOSITORY_NAME="spotify-track-db"
CLUSTER_NAME="${CLUSTER_NAME:-spotify-cluster}"  # Default cluster name, can be overridden
SERVICE_NAME="spotify-track-db"
TASK_FAMILY="spotify-track-db"

# Step 1: Get AWS Account ID
echo -e "${YELLOW}📋 Step 1: Getting AWS Account ID...${NC}"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
if [ -z "$AWS_ACCOUNT_ID" ]; then
    echo -e "${RED}❌ Failed to get AWS Account ID. Make sure AWS CLI is configured.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ AWS Account ID: $AWS_ACCOUNT_ID${NC}\n"

# Step 2: Check if ECR repository exists, create if it doesn't
echo -e "${YELLOW}📦 Step 2: Checking ECR repository...${NC}"
if aws ecr describe-repositories --repository-names $REPOSITORY_NAME --region $REGION > /dev/null 2>&1; then
    echo -e "${GREEN}✅ ECR repository '$REPOSITORY_NAME' already exists${NC}\n"
else
    echo -e "${YELLOW}Creating ECR repository '$REPOSITORY_NAME'...${NC}"
    aws ecr create-repository \
        --repository-name $REPOSITORY_NAME \
        --region $REGION \
        --image-scanning-configuration scanOnPush=true
    echo -e "${GREEN}✅ ECR repository created${NC}\n"
fi

# Step 3: Authenticate Docker to ECR
echo -e "${YELLOW}🔐 Step 3: Authenticating Docker to ECR...${NC}"
aws ecr get-login-password --region $REGION | \
    docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com
echo -e "${GREEN}✅ Docker authenticated to ECR${NC}\n"

# Step 4: Build Docker image for linux/amd64 (Fargate compatibility)
echo -e "${YELLOW}🏗️  Step 4: Building Docker image for linux/amd64...${NC}"
docker buildx build --platform linux/amd64 -t $REPOSITORY_NAME:latest .
echo -e "${GREEN}✅ Docker image built successfully${NC}\n"

# Step 5: Tag the image
echo -e "${YELLOW}🏷️  Step 5: Tagging Docker image...${NC}"
IMAGE_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/${REPOSITORY_NAME}:latest"
docker tag ${REPOSITORY_NAME}:latest $IMAGE_URI
echo -e "${GREEN}✅ Image tagged: $IMAGE_URI${NC}\n"

# Step 6: Push to ECR
echo -e "${YELLOW}⬆️  Step 6: Pushing image to ECR...${NC}"
docker push $IMAGE_URI
echo -e "${GREEN}✅ Image pushed to ECR${NC}\n"

# Step 7: Update task definition with actual account ID
echo -e "${YELLOW}📝 Step 7: Updating task definition...${NC}"
TEMP_TASK_DEF=$(mktemp)
sed "s/YOUR_ACCOUNT_ID/$AWS_ACCOUNT_ID/g" ecs-task-definition.json > $TEMP_TASK_DEF
echo -e "${GREEN}✅ Task definition updated${NC}\n"

# Step 8: Register new task definition
echo -e "${YELLOW}📋 Step 8: Registering task definition...${NC}"
TASK_REVISION=$(aws ecs register-task-definition \
    --cli-input-json file://$TEMP_TASK_DEF \
    --region $REGION \
    --query 'taskDefinition.revision' \
    --output text)
rm $TEMP_TASK_DEF
echo -e "${GREEN}✅ Task definition registered: $TASK_FAMILY:$TASK_REVISION${NC}\n"

# Step 9: Check if CloudWatch log group exists
echo -e "${YELLOW}📊 Step 9: Checking CloudWatch log group...${NC}"
if aws logs describe-log-groups --log-group-name-prefix "/ecs/$REPOSITORY_NAME" --region $REGION | grep -q "/ecs/$REPOSITORY_NAME"; then
    echo -e "${GREEN}✅ CloudWatch log group exists${NC}\n"
else
    echo -e "${YELLOW}Creating CloudWatch log group...${NC}"
    aws logs create-log-group \
        --log-group-name /ecs/$REPOSITORY_NAME \
        --region $REGION
    echo -e "${GREEN}✅ CloudWatch log group created${NC}\n"
fi

# Step 10: Check if ECS cluster exists
echo -e "${YELLOW}🔍 Step 10: Checking ECS cluster...${NC}"
if aws ecs describe-clusters --clusters $CLUSTER_NAME --region $REGION --query 'clusters[0].status' --output text | grep -q "ACTIVE"; then
    echo -e "${GREEN}✅ ECS cluster '$CLUSTER_NAME' exists${NC}\n"
else
    echo -e "${RED}❌ ECS cluster '$CLUSTER_NAME' does not exist.${NC}"
    echo -e "${YELLOW}Creating cluster...${NC}"
    aws ecs create-cluster --cluster-name $CLUSTER_NAME --region $REGION
    echo -e "${GREEN}✅ ECS cluster created${NC}\n"
fi

# Step 11: Check if service exists and update or create
echo -e "${YELLOW}🔄 Step 11: Updating or creating ECS service...${NC}"
if aws ecs describe-services --cluster $CLUSTER_NAME --services $SERVICE_NAME --region $REGION --query 'services[0].status' --output text 2>/dev/null | grep -qE "ACTIVE|DRAINING"; then
    echo -e "${YELLOW}Service exists. Updating with new task definition...${NC}"
    aws ecs update-service \
        --cluster $CLUSTER_NAME \
        --service $SERVICE_NAME \
        --task-definition $TASK_FAMILY:$TASK_REVISION \
        --force-new-deployment \
        --region $REGION \
        --query 'service.serviceName' \
        --output text
    echo -e "${GREEN}✅ Service updated successfully${NC}\n"
else
    echo -e "${YELLOW}Service does not exist. You need to create it manually with proper VPC configuration.${NC}"
    echo -e "${YELLOW}Run the following command (update subnet and security group):${NC}\n"
    echo -e "${BLUE}aws ecs create-service \\
    --cluster $CLUSTER_NAME \\
    --service-name $SERVICE_NAME \\
    --task-definition $TASK_FAMILY:$TASK_REVISION \\
    --desired-count 1 \\
    --launch-type FARGATE \\
    --network-configuration \"awsvpcConfiguration={subnets=[YOUR_SUBNET_ID],securityGroups=[YOUR_SG_ID],assignPublicIp=ENABLED}\" \\
    --region $REGION${NC}\n"
fi

# Step 12: Display deployment info
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}\n"
echo -e "${BLUE}📊 Deployment Information:${NC}"
echo -e "  Region: ${YELLOW}$REGION${NC}"
echo -e "  Cluster: ${YELLOW}$CLUSTER_NAME${NC}"
echo -e "  Service: ${YELLOW}$SERVICE_NAME${NC}"
echo -e "  Task Definition: ${YELLOW}$TASK_FAMILY:$TASK_REVISION${NC}"
echo -e "  Image: ${YELLOW}$IMAGE_URI${NC}\n"

echo -e "${BLUE}🔍 View logs:${NC}"
echo -e "  aws logs tail /ecs/$REPOSITORY_NAME --follow --region $REGION\n"

echo -e "${BLUE}📋 View service:${NC}"
echo -e "  aws ecs describe-services --cluster $CLUSTER_NAME --services $SERVICE_NAME --region $REGION\n"

echo -e "${BLUE}🔄 View tasks:${NC}"
echo -e "  aws ecs list-tasks --cluster $CLUSTER_NAME --service-name $SERVICE_NAME --region $REGION\n"

echo -e "${GREEN}🎉 Deployment script completed successfully!${NC}"
