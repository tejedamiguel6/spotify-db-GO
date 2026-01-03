# 🚀 AWS ECS Deployment Guide

Complete guide to deploy Spotify Track DB to AWS ECS (Fargate).

## 📋 Prerequisites

Before deploying, ensure you have:

- ✅ **AWS CLI** installed and configured (`aws configure`)
- ✅ **Docker** installed and running
- ✅ **AWS Account** with appropriate permissions
- ✅ **[.env](.env)** file with your credentials (use `.env.example` as template)

## 🎯 Quick Start (Automated Deployment)

### Step 1: Set Up AWS Secrets
Run this **once** to create secrets in AWS Secrets Manager:

```bash
./setup-aws-secrets.sh
```

This script will:
- Read your `.env` file
- Create/update secrets in AWS Secrets Manager:
  - `spotify-track-db/DATABASE_URL`
  - `spotify-track-db/SPOTIFY_CLIENT_ID`
  - `spotify-track-db/SPOTIFY_CLIENT_SECRET`

### Step 2: Deploy to ECS
Run this to build, push, and deploy:

```bash
./deploy-to-ecs.sh
```

This script will:
1. ✅ Get your AWS Account ID
2. ✅ Create ECR repository (if needed)
3. ✅ Authenticate Docker to ECR
4. ✅ Build Docker image
5. ✅ Push image to ECR
6. ✅ Update task definition
7. ✅ Register new task definition
8. ✅ Create CloudWatch log group
9. ✅ Update ECS service (or show create command)

### Step 3: First Time Setup (If Service Doesn't Exist)

If you're deploying for the first time, you'll need to create the ECS service manually with your VPC configuration:

```bash
# Get your VPC details first
aws ec2 describe-vpcs --region us-east-1
aws ec2 describe-subnets --region us-east-1
aws ec2 describe-security-groups --region us-east-1

# Create the service (replace placeholders)
aws ecs create-service \
    --cluster spotify-cluster \
    --service-name spotify-track-db \
    --task-definition spotify-track-db \
    --desired-count 1 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxxx],securityGroups=[sg-xxxxx],assignPublicIp=ENABLED}" \
    --region us-east-1
```

---

## 🔧 Manual Deployment (Step-by-Step)

If you prefer to run commands manually:

### 1. Get AWS Account ID
```bash
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
echo $AWS_ACCOUNT_ID
```

### 2. Create ECR Repository
```bash
aws ecr create-repository \
    --repository-name spotify-track-db \
    --region us-east-1
```

### 3. Login to ECR
```bash
aws ecr get-login-password --region us-east-1 | \
    docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com
```

### 4. Build Docker Image
```bash
docker build -t spotify-track-db:latest .
```

### 5. Tag and Push
```bash
docker tag spotify-track-db:latest ${AWS_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/spotify-track-db:latest
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/spotify-track-db:latest
```

### 6. Update Task Definition
Replace `YOUR_ACCOUNT_ID` in [ecs-task-definition.json](ecs-task-definition.json):

```bash
sed -i '' "s/YOUR_ACCOUNT_ID/${AWS_ACCOUNT_ID}/g" ecs-task-definition.json
```

### 7. Register Task Definition
```bash
aws ecs register-task-definition \
    --cli-input-json file://ecs-task-definition.json \
    --region us-east-1
```

### 8. Update Service
```bash
aws ecs update-service \
    --cluster spotify-cluster \
    --service spotify-track-db \
    --task-definition spotify-track-db \
    --force-new-deployment \
    --region us-east-1
```

---

## 📊 Monitoring & Debugging

### View Logs
```bash
# Tail logs in real-time
aws logs tail /ecs/spotify-track-db --follow --region us-east-1

# View specific time range
aws logs tail /ecs/spotify-track-db --since 1h --region us-east-1
```

### Check Service Status
```bash
aws ecs describe-services \
    --cluster spotify-cluster \
    --services spotify-track-db \
    --region us-east-1
```

### List Running Tasks
```bash
aws ecs list-tasks \
    --cluster spotify-cluster \
    --service-name spotify-track-db \
    --region us-east-1
```

### Describe Task (Get Details)
```bash
# Get task ARN first
TASK_ARN=$(aws ecs list-tasks --cluster spotify-cluster --service-name spotify-track-db --region us-east-1 --query 'taskArns[0]' --output text)

# Describe the task
aws ecs describe-tasks \
    --cluster spotify-cluster \
    --tasks $TASK_ARN \
    --region us-east-1
```

### Stop Running Task (Force Restart)
```bash
TASK_ARN=$(aws ecs list-tasks --cluster spotify-cluster --service-name spotify-track-db --region us-east-1 --query 'taskArns[0]' --output text)

aws ecs stop-task \
    --cluster spotify-cluster \
    --task $TASK_ARN \
    --region us-east-1
```

---

## 🔐 IAM Permissions Required

Your ECS Task Execution Role needs these permissions:

### Policy for Secrets Manager Access
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-east-1:*:secret:spotify-track-db/*"
      ]
    }
  ]
}
```

### Policy for CloudWatch Logs
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:us-east-1:*:log-group:/ecs/spotify-track-db:*"
    }
  ]
}
```

---

## 🛠️ Common Issues & Solutions

### Issue: Task Fails to Start
**Check:**
- CloudWatch logs for error messages
- Task execution role has correct permissions
- Secrets exist in Secrets Manager
- Security group allows outbound traffic

```bash
aws logs tail /ecs/spotify-track-db --region us-east-1
```

### Issue: Can't Connect to Database
**Check:**
- DATABASE_URL secret is correct
- Security group allows connection to database
- Database is accessible from ECS subnet

### Issue: Image Pull Errors
**Check:**
- ECR repository exists
- Task execution role has ECR permissions
- Image was pushed successfully

```bash
aws ecr describe-images --repository-name spotify-track-db --region us-east-1
```

---

## 💰 Cost Estimation

Approximate monthly costs for running on Fargate:

- **Fargate** (0.25 vCPU, 0.5 GB RAM, 24/7): ~$12/month
- **CloudWatch Logs** (minimal): ~$1/month
- **Data Transfer**: Varies
- **Secrets Manager**: $0.40/secret/month = $1.20/month

**Total**: ~$15-20/month

---

## 🔄 Updating Your Application

After making code changes:

1. **Commit your changes:**
   ```bash
   git add .
   git commit -m "Your changes"
   git push
   ```

2. **Redeploy:**
   ```bash
   ./deploy-to-ecs.sh
   ```

The script will automatically:
- Build new Docker image
- Push to ECR
- Force new deployment with latest image

---

## 🎯 Environment Variables

Set these in your shell to customize deployment:

```bash
# Override default cluster name
export CLUSTER_NAME="my-custom-cluster"

# Then deploy
./deploy-to-ecs.sh
```

---

## 📚 Additional Resources

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS Fargate Pricing](https://aws.amazon.com/fargate/pricing/)
- [ECR User Guide](https://docs.aws.amazon.com/ecr/)
- [Secrets Manager](https://docs.aws.amazon.com/secretsmanager/)

---

**Happy Deploying! 🚀**
