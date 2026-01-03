# AWS Deployment Guide for Developers

A comprehensive guide to deploying your Go application to AWS ECS, written for developers new to AWS.

---

## Table of Contents
1. [Overview: What is AWS ECS?](#overview-what-is-aws-ecs)
2. [Architecture: Understanding the Components](#architecture-understanding-the-components)
3. [Step-by-Step Deployment Process](#step-by-step-deployment-process)
4. [What We Built in This Deployment](#what-we-built-in-this-deployment)
5. [How Everything Works Together](#how-everything-works-together)
6. [Common AWS Concepts Explained](#common-aws-concepts-explained)
7. [Managing Your Deployment](#managing-your-deployment)
8. [Costs & Billing](#costs--billing)
9. [Troubleshooting](#troubleshooting)

---

## Overview: What is AWS ECS?

**ECS (Elastic Container Service)** is AWS's container orchestration platform - think of it as a service that runs your Docker containers 24/7 in the cloud.

### Why Use ECS?
- **Always Running**: Your app runs continuously, even when your laptop is off
- **Scalable**: Can automatically handle more traffic
- **Managed**: AWS handles server maintenance, updates, and infrastructure
- **Production-Ready**: Used by companies of all sizes

### ECS vs. Other Options

| Service | What It Is | Best For |
|---------|-----------|----------|
| **ECS Fargate** | Serverless containers (what we used) | Apps that need to run 24/7 without managing servers |
| **EC2** | Virtual servers you manage yourself | Full control over infrastructure |
| **Lambda** | Run code on-demand, no servers | Event-driven tasks, APIs with sporadic traffic |
| **Heroku/Railway** | Platform-as-a-Service (PaaS) | Quick deployments, less control |

---

## Architecture: Understanding the Components

Here's what we built for your Spotify Track DB:

```
┌─────────────────────────────────────────────────────────────┐
│                         INTERNET                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │ Security Group  │ ◄── Firewall rules (port 8080 open)
              │ sg-071cb58cda... │
              └────────┬─────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   ECS Service   │ ◄── Manages your running containers
              │ spotify-track-db │
              └────────┬─────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   ECS Task      │ ◄── Your running container instance
              │  (Fargate)      │
              └────────┬─────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
    ┌────────┐   ┌──────────┐  ┌──────────────┐
    │  ECR   │   │ Secrets  │  │ CloudWatch   │
    │ (Image)│   │ Manager  │  │ (Logs)       │
    └────────┘   └──────────┘  └──────────────┘
                      │
                      ▼
                ┌──────────────┐
                │ Neon Database│
                │ (PostgreSQL) │
                └──────────────┘
```

### Component Breakdown

#### 1. **ECR (Elastic Container Registry)**
- **What**: Docker image storage (like Docker Hub, but AWS-hosted)
- **Purpose**: Stores your built Docker images
- **In Our Setup**: `748344702309.dkr.ecr.us-east-2.amazonaws.com/spotify-track-db`
- **Analogy**: Think of it as GitHub, but for Docker images instead of code

#### 2. **ECS Cluster**
- **What**: A logical grouping of services and tasks
- **Purpose**: Organizes your containerized applications
- **In Our Setup**: `spotify-cluster` in `us-east-2`
- **Analogy**: Like a folder that contains all your running apps

#### 3. **ECS Service**
- **What**: Ensures your desired number of tasks are always running
- **Purpose**: Maintains exactly 1 running instance of your app (in our case)
- **In Our Setup**: `spotify-track-db` service running 1 task
- **Analogy**: Like PM2 or systemd - automatically restarts your app if it crashes

#### 4. **ECS Task**
- **What**: A running instance of your container
- **Purpose**: Your actual application executing in the cloud
- **In Our Setup**: Runs your Go server on port 8080
- **Analogy**: A single instance of `node index.js` or `go run main.go`

#### 5. **Task Definition**
- **What**: A JSON blueprint describing how to run your container
- **Purpose**: Specifies CPU, memory, environment variables, secrets, etc.
- **In Our Setup**: `ecs-task-definition.json` - 0.25 vCPU, 512 MB RAM
- **Analogy**: Like a `docker-compose.yml` file

#### 6. **Fargate**
- **What**: Serverless compute engine for containers
- **Purpose**: AWS manages the servers for you
- **In Our Setup**: Runs your container without you managing EC2 instances
- **Analogy**: Like serverless functions (Lambda), but for long-running containers

#### 7. **IAM Roles**
- **What**: Permissions that define what AWS resources can access
- **Purpose**: Security - containers can only do what they're allowed to
- **In Our Setup**:
  - `ecsTaskExecutionRole`: Allows ECS to pull images from ECR and access secrets
  - `ecsTaskRole`: Permissions your application has while running
- **Analogy**: Like user permissions on Linux (`sudo` vs regular user)

#### 8. **Security Group**
- **What**: Virtual firewall for your container
- **Purpose**: Controls inbound and outbound network traffic
- **In Our Setup**: `sg-071cb58cda54a5995` - allows port 8080 from anywhere
- **Analogy**: Like `iptables` or your home router's port forwarding rules

#### 9. **VPC (Virtual Private Cloud)**
- **What**: Your isolated network in AWS
- **Purpose**: Network isolation and security
- **In Our Setup**: Default VPC with 3 subnets across availability zones
- **Analogy**: Your own private data center network

#### 10. **AWS Secrets Manager**
- **What**: Encrypted storage for sensitive data
- **Purpose**: Securely stores API keys, passwords, database URLs
- **In Our Setup**: Stores `DATABASE_URL`, `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`
- **Analogy**: Like a password manager (1Password, LastPass), but for your apps

#### 11. **CloudWatch Logs**
- **What**: Log aggregation service
- **Purpose**: Collects and stores your application logs
- **In Our Setup**: `/ecs/spotify-track-db` log group
- **Analogy**: Like `tail -f` for your cloud app, or Papertrail/Loggly

---

## Step-by-Step Deployment Process

Here's exactly what we did, in order:

### Phase 1: Security & Preparation

#### Step 1: Security Audit
```bash
# What we checked:
- No credentials in source code ✓
- .env file properly gitignored ✓
- Created .env.example for other developers ✓
```

**Why**: Prevent accidentally committing secrets to GitHub

---

### Phase 2: AWS Secrets Setup

#### Step 2: Create AWS Secrets Manager Secrets
```bash
./setup-aws-secrets.sh
```

**What this does**:
1. Reads your local `.env` file
2. Creates 3 secrets in AWS Secrets Manager:
   - `spotify-track-db/DATABASE_URL`
   - `spotify-track-db/SPOTIFY_CLIENT_ID`
   - `spotify-track-db/SPOTIFY_CLIENT_SECRET`

**Why**: Keeps credentials out of your Docker image. ECS injects these as environment variables at runtime.

**Behind the scenes**:
```bash
aws secretsmanager create-secret \
    --name "spotify-track-db/DATABASE_URL" \
    --secret-string "postgresql://..." \
    --region us-east-2
```

---

### Phase 3: Docker Image Creation

#### Step 3: Build Docker Image
```bash
# What the deployment script does:
docker buildx build --platform linux/amd64 -t spotify-track-db:latest .
```

**Key Point**: `--platform linux/amd64` is critical!
- Your Mac uses **ARM64** (Apple Silicon)
- AWS Fargate uses **AMD64** (Intel x86_64)
- Without specifying platform, the image won't work on AWS

**The Dockerfile** (multi-stage build):
```dockerfile
# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /server .
EXPOSE 8080
CMD ["./server"]
```

**Why multi-stage**:
- Stage 1: Includes Go compiler and build tools (~800MB)
- Stage 2: Only includes the compiled binary (~20MB)
- Result: Much smaller, faster-to-deploy image

---

### Phase 4: Push to AWS

#### Step 4: Authenticate to ECR
```bash
aws ecr get-login-password --region us-east-2 | \
    docker login --username AWS --password-stdin \
    748344702309.dkr.ecr.us-east-2.amazonaws.com
```

**What this does**: Logs Docker into AWS's private registry (like `docker login` for Docker Hub)

#### Step 5: Tag and Push Image
```bash
# Tag the image with ECR repository URL
docker tag spotify-track-db:latest \
    748344702309.dkr.ecr.us-east-2.amazonaws.com/spotify-track-db:latest

# Push to ECR
docker push 748344702309.dkr.ecr.us-east-2.amazonaws.com/spotify-track-db:latest
```

**What this does**: Uploads your Docker image to AWS so ECS can use it

---

### Phase 5: IAM Permissions

#### Step 6: Create IAM Roles

**Role 1: ecsTaskExecutionRole** (what ECS needs to start your container)
```bash
aws iam create-role --role-name ecsTaskExecutionRole \
    --assume-role-policy-document '{
        "Version": "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Principal": {"Service": "ecs-tasks.amazonaws.com"},
            "Action": "sts:AssumeRole"
        }]
    }'
```

**Attach permissions**:
```bash
# Permission to pull from ECR
aws iam attach-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

# Permission to read secrets
aws iam put-role-policy \
    --role-name ecsTaskExecutionRole \
    --policy-name SecretsManagerAccess \
    --policy-document '{
        "Version": "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Action": ["secretsmanager:GetSecretValue"],
            "Resource": ["arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/*"]
        }]
    }'
```

**Role 2: ecsTaskRole** (what your app can do while running)
```bash
aws iam create-role --role-name ecsTaskRole \
    --assume-role-policy-document '{
        "Version": "2012-10-17",
        "Statement": [{
            "Effect": "Allow",
            "Principal": {"Service": "ecs-tasks.amazonaws.com"},
            "Action": "sts:AssumeRole"
        }]
    }'
```

**Think of it this way**:
- **ecsTaskExecutionRole**: "Backstage pass" for ECS to set up your container
- **ecsTaskRole**: "VIP pass" for your running app to access other AWS services

---

### Phase 6: ECS Setup

#### Step 7: Create ECS Cluster
```bash
aws ecs create-cluster --cluster-name spotify-cluster --region us-east-2
```

**What this does**: Creates a logical grouping for your services

#### Step 8: Register Task Definition
```bash
aws ecs register-task-definition \
    --cli-input-json file://ecs-task-definition.json \
    --region us-east-2
```

**What's in the task definition** (`ecs-task-definition.json`):
```json
{
  "family": "spotify-track-db",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",        // 0.25 vCPU
  "memory": "512",     // 512 MB RAM
  "executionRoleArn": "arn:aws:iam::748344702309:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::748344702309:role/ecsTaskRole",
  "containerDefinitions": [{
    "name": "spotify-track-db",
    "image": "748344702309.dkr.ecr.us-east-2.amazonaws.com/spotify-track-db:latest",
    "portMappings": [{"containerPort": 8080}],
    "secrets": [
      {
        "name": "DATABASE_URL",
        "valueFrom": "arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/DATABASE_URL"
      },
      {
        "name": "SPOTIFY_CLIENT_ID",
        "valueFrom": "arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/SPOTIFY_CLIENT_ID"
      },
      {
        "name": "SPOTIFY_CLIENT_SECRET",
        "valueFrom": "arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/SPOTIFY_CLIENT_SECRET"
      }
    ],
    "logConfiguration": {
      "logDriver": "awslogs",
      "options": {
        "awslogs-group": "/ecs/spotify-track-db",
        "awslogs-region": "us-east-2",
        "awslogs-stream-prefix": "ecs"
      }
    }
  }]
}
```

**Key parts explained**:
- `cpu: "256"`: 0.25 vCPU (1024 = 1 full vCPU)
- `memory: "512"`: 512 MB RAM
- `secrets`: ECS fetches these from Secrets Manager and injects as env vars
- `logConfiguration`: Sends logs to CloudWatch

#### Step 9: Create ECS Service
```bash
aws ecs create-service \
    --cluster spotify-cluster \
    --service-name spotify-track-db \
    --task-definition spotify-track-db \
    --desired-count 1 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={
        subnets=[subnet-06d21d97bb8da1d7d,subnet-08bad66d1ab44e030,subnet-07dfb78e9905054af],
        securityGroups=[sg-071cb58cda54a5995],
        assignPublicIp=ENABLED
    }" \
    --region us-east-2
```

**What this does**:
- Creates a service that maintains 1 running task
- If the task crashes, ECS automatically starts a new one
- Assigns a public IP so you can access it from the internet
- Uses 3 subnets across different availability zones for reliability

---

### Phase 7: Networking & Security

#### Step 10: Configure Security Group
```bash
aws ec2 authorize-security-group-ingress \
    --group-id sg-071cb58cda54a5995 \
    --protocol tcp \
    --port 8080 \
    --cidr 0.0.0.0/0 \
    --region us-east-2
```

**What this does**: Opens port 8080 to the internet

**Security Group Rules**:
- **Inbound**: Port 8080 from anywhere (0.0.0.0/0)
- **Outbound**: All traffic allowed (to access Neon database, Spotify API, etc.)

**Why we needed this**: By default, security groups block all inbound traffic. Without this rule, your API would be unreachable.

---

## What We Built in This Deployment

### Infrastructure Created

| Resource | Name/ID | Purpose |
|----------|---------|---------|
| **ECR Repository** | `spotify-track-db` | Stores Docker images |
| **ECS Cluster** | `spotify-cluster` | Groups services |
| **ECS Service** | `spotify-track-db` | Manages task lifecycle |
| **ECS Task Definition** | `spotify-track-db:1` | Container blueprint |
| **IAM Role (Execution)** | `ecsTaskExecutionRole` | ECS permissions |
| **IAM Role (Task)** | `ecsTaskRole` | App permissions |
| **Security Group** | `sg-071cb58cda54a5995` | Firewall rules |
| **Secrets** | 3 secrets in Secrets Manager | Secure credential storage |
| **CloudWatch Log Group** | `/ecs/spotify-track-db` | Application logs |

### Files Created Locally

| File | Purpose |
|------|---------|
| `Dockerfile` | Defines how to build the Docker image |
| `ecs-task-definition.json` | ECS task configuration |
| `deploy-to-ecs.sh` | Automated deployment script |
| `setup-aws-secrets.sh` | Secrets setup script |
| `.env.example` | Template for environment variables |
| `DEPLOYMENT.md` | Deployment documentation |

---

## How Everything Works Together

### Deployment Flow

```
1. Developer runs: ./deploy-to-ecs.sh
                     ↓
2. Script builds Docker image (for linux/amd64)
                     ↓
3. Image pushed to ECR
                     ↓
4. Task definition registered with ECS
                     ↓
5. ECS Service triggers new deployment
                     ↓
6. ECS pulls image from ECR
                     ↓
7. ECS fetches secrets from Secrets Manager
                     ↓
8. Task starts running on Fargate
                     ↓
9. Health checks pass, traffic routed to new task
                     ↓
10. Old task (if any) is stopped
```

### Runtime Flow (When a Request Comes In)

```
1. User makes request: http://18.219.72.41:8080/now-listening-to
                     ↓
2. Request hits AWS network
                     ↓
3. Security Group checks: "Is port 8080 allowed?" → Yes ✓
                     ↓
4. Traffic forwarded to ECS Task
                     ↓
5. Your Go app handles the request
                     ↓
6. App queries Neon database (using DATABASE_URL from Secrets Manager)
                     ↓
7. App calls Spotify API (using credentials from Secrets Manager)
                     ↓
8. Response returned to user
                     ↓
9. Request logged to CloudWatch
```

### Environment Variable Injection

When your container starts:

```
1. ECS reads task definition
2. Sees "secrets" section
3. Calls Secrets Manager API to fetch values
4. Injects as environment variables into container
5. Your app reads os.Getenv("DATABASE_URL")
```

**In your code** (`internal/repository/db.go`):
```go
// This works because ECS injects secrets as environment variables
dsn := os.Getenv("DATABASE_URL")
pool, err := pgxpool.New(context.Background(), dsn)
```

---

## Common AWS Concepts Explained

### Regions and Availability Zones

**Region**: `us-east-2` (Ohio)
- A geographic area with multiple data centers
- Your entire deployment lives in one region

**Availability Zones (AZs)**: `us-east-2a`, `us-east-2b`, `us-east-2c`
- Independent data centers within a region
- Your service spans 3 AZs for high availability
- If one AZ goes down, your app keeps running in the others

### ARNs (Amazon Resource Names)

Think of ARNs as unique IDs for AWS resources:

```
arn:aws:ecs:us-east-2:748344702309:service/spotify-cluster/spotify-track-db
│   │   │   │          │              │
│   │   │   │          │              └─ Resource name
│   │   │   │          └─ AWS Account ID
│   │   │   └─ Region
│   │   └─ Service (ECS)
│   └─ Partition (aws, aws-cn, aws-us-gov)
└─ Always starts with "arn"
```

### Tags

Tags are key-value pairs for organizing resources:
```json
{
  "Environment": "production",
  "Project": "spotify-track-db",
  "ManagedBy": "terraform"
}
```

We didn't add tags in this deployment, but it's a best practice for production.

---

## Managing Your Deployment

### View Logs
```bash
# Real-time logs (like `tail -f`)
aws logs tail /ecs/spotify-track-db --follow --region us-east-2

# Last hour of logs
aws logs tail /ecs/spotify-track-db --since 1h --region us-east-2

# Search for errors
aws logs tail /ecs/spotify-track-db --filter-pattern "ERROR" --region us-east-2
```

### Check Service Status
```bash
aws ecs describe-services \
    --cluster spotify-cluster \
    --services spotify-track-db \
    --region us-east-2 \
    --query 'services[0].{Running:runningCount,Desired:desiredCount,Status:status}'
```

Output:
```
Running: 1
Desired: 1
Status: ACTIVE
```

### Get Public IP
```bash
# Get the running task
TASK_ARN=$(aws ecs list-tasks \
    --cluster spotify-cluster \
    --service-name spotify-track-db \
    --region us-east-2 \
    --query 'taskArns[0]' \
    --output text)

# Get network interface ID
ENI_ID=$(aws ecs describe-tasks \
    --cluster spotify-cluster \
    --tasks $TASK_ARN \
    --region us-east-2 \
    --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' \
    --output text)

# Get public IP
aws ec2 describe-network-interfaces \
    --network-interface-ids $ENI_ID \
    --region us-east-2 \
    --query 'NetworkInterfaces[0].Association.PublicIp' \
    --output text
```

### Update Your App (Deploy New Version)

After making code changes:

```bash
# 1. Commit changes to git
git add .
git commit -m "Your changes"

# 2. Run deployment script
./deploy-to-ecs.sh
```

The script will:
1. Build new Docker image
2. Push to ECR with `:latest` tag
3. Force ECS to deploy new version

**ECS Rolling Update**:
- Starts new task with new image
- Waits for it to be healthy
- Stops old task
- Zero downtime!

### Scale Your Service

```bash
# Run 3 instances instead of 1
aws ecs update-service \
    --cluster spotify-cluster \
    --service spotify-track-db \
    --desired-count 3 \
    --region us-east-2
```

**Note**: With multiple tasks, you'd want a load balancer (not set up yet).

### Stop Your Service (Save Money)

```bash
# Set to 0 tasks
aws ecs update-service \
    --cluster spotify-cluster \
    --service spotify-track-db \
    --desired-count 0 \
    --region us-east-2
```

Your service still exists, but no tasks are running (no charges for Fargate).

### Completely Delete Everything

```bash
# 1. Delete service
aws ecs delete-service \
    --cluster spotify-cluster \
    --service spotify-track-db \
    --force \
    --region us-east-2

# 2. Delete cluster
aws ecs delete-cluster \
    --cluster spotify-cluster \
    --region us-east-2

# 3. Delete ECR images
aws ecr delete-repository \
    --repository-name spotify-track-db \
    --force \
    --region us-east-2

# 4. Delete secrets (optional - be careful!)
aws secretsmanager delete-secret \
    --secret-id spotify-track-db/DATABASE_URL \
    --force-delete-without-recovery \
    --region us-east-2
```

---

## Costs & Billing

### Current Monthly Costs (Estimated)

| Service | Usage | Cost |
|---------|-------|------|
| **Fargate** | 0.25 vCPU, 0.5 GB RAM, 24/7 | ~$12/month |
| **CloudWatch Logs** | ~1 GB/month | ~$0.50/month |
| **Secrets Manager** | 3 secrets | $1.20/month ($0.40/secret) |
| **ECR Storage** | ~100 MB | ~$0.01/month |
| **Data Transfer** | Outbound to internet | Varies (first 100 GB free) |
| **Total** | | **~$14-15/month** |

### How Fargate Pricing Works

**Pricing Formula**:
```
Cost = (vCPU hours × $0.04048) + (GB hours × $0.004445)
```

**For 0.25 vCPU + 0.5 GB RAM running 24/7**:
```
vCPU cost = 0.25 × 730 hours × $0.04048 = $7.39/month
Memory cost = 0.5 × 730 hours × $0.004445 = $1.62/month
Total = $9.01/month
```

(Prices vary slightly by region; us-east-2 used above)

### Ways to Reduce Costs

1. **Run only when needed** (not 24/7):
   ```bash
   # Stop at night, start in morning
   aws ecs update-service --desired-count 0  # Stop
   aws ecs update-service --desired-count 1  # Start
   ```

2. **Use smaller instance sizes**:
   - Current: 0.25 vCPU, 0.5 GB RAM
   - Minimum: 0.25 vCPU, 0.5 GB RAM (you're already at minimum)

3. **Optimize Docker image**:
   - Smaller images = faster deployments = less data transfer
   - We're already using multi-stage builds (good!)

4. **Use AWS Free Tier** (first 12 months):
   - Fargate: First 6 months get some free tier
   - CloudWatch: 5 GB logs free/month
   - Secrets Manager: 30-day trial, then $0.40/secret/month

### Setting Up Billing Alerts

```bash
# Create SNS topic for alerts
aws sns create-topic --name billing-alerts --region us-east-1

# Subscribe your email
aws sns subscribe \
    --topic-arn arn:aws:sns:us-east-1:748344702309:billing-alerts \
    --protocol email \
    --notification-endpoint your-email@example.com
```

Then create a CloudWatch alarm in the AWS Console:
- Go to CloudWatch → Billing → Create Alarm
- Set threshold (e.g., $20/month)
- Choose SNS topic created above

---

## Troubleshooting

### Common Issues

#### 1. Container Keeps Restarting

**Check logs**:
```bash
aws logs tail /ecs/spotify-track-db --since 10m --region us-east-2
```

**Common causes**:
- App crashes on startup (check logs for errors)
- Database connection fails (check DATABASE_URL secret)
- Port conflict (make sure app listens on 8080)

**How to debug**:
```bash
# See task stopped reason
aws ecs describe-tasks \
    --cluster spotify-cluster \
    --tasks <TASK_ID> \
    --region us-east-2 \
    --query 'tasks[0].stoppedReason'
```

#### 2. Can't Access API (Connection Timeout)

**Checklist**:
- [ ] Security group allows port 8080 inbound
- [ ] Task has public IP assigned
- [ ] Task is in RUNNING state
- [ ] App is listening on 0.0.0.0:8080 (not 127.0.0.1)

**Fix security group**:
```bash
aws ec2 authorize-security-group-ingress \
    --group-id sg-071cb58cda54a5995 \
    --protocol tcp \
    --port 8080 \
    --cidr 0.0.0.0/0 \
    --region us-east-2
```

**Check task status**:
```bash
aws ecs describe-services \
    --cluster spotify-cluster \
    --services spotify-track-db \
    --region us-east-2 \
    --query 'services[0].events[0:3]'
```

#### 3. "Image Not Found" or "Cannot Pull Container"

**Causes**:
- Wrong image tag or repository name
- ecsTaskExecutionRole doesn't have ECR permissions
- Image was built for wrong platform (arm64 vs amd64)

**Solutions**:
```bash
# Check if image exists
aws ecr describe-images \
    --repository-name spotify-track-db \
    --region us-east-2

# Rebuild with correct platform
docker buildx build --platform linux/amd64 -t spotify-track-db:latest .
```

#### 4. "Unable to Assume Role" Error

**Cause**: IAM roles don't have correct trust relationships

**Fix**:
```bash
# Check role trust policy
aws iam get-role --role-name ecsTaskExecutionRole \
    --query 'Role.AssumeRolePolicyDocument'

# Should allow "ecs-tasks.amazonaws.com"
```

#### 5. Secrets Not Loading

**Symptoms**: App gets empty environment variables

**Checklist**:
- [ ] Secrets exist in Secrets Manager
- [ ] Secret ARNs in task definition are correct
- [ ] ecsTaskExecutionRole has `secretsmanager:GetSecretValue` permission
- [ ] Secrets are in the same region (us-east-2)

**Verify secrets**:
```bash
aws secretsmanager get-secret-value \
    --secret-id spotify-track-db/DATABASE_URL \
    --region us-east-2
```

---

## Next Steps & Production Improvements

### Things We Haven't Done Yet (But Should for Production)

#### 1. **Application Load Balancer (ALB)**
Currently, your app gets a new public IP every time the task restarts.

**Benefits**:
- Stable URL/domain name
- HTTPS/SSL support
- Health checks
- Ability to run multiple tasks

**Cost**: ~$16/month + $0.008 per GB of data processed

#### 2. **Custom Domain with Route 53**
Instead of `http://18.219.72.41:8080`, use `https://api.yoursite.com`

**Steps**:
1. Buy domain in Route 53 (~$12/year for .com)
2. Create ALB
3. Request SSL certificate from AWS Certificate Manager (free!)
4. Point domain to ALB

#### 3. **Auto-Scaling**
Automatically scale based on CPU/memory usage

```bash
aws application-autoscaling register-scalable-target \
    --service-namespace ecs \
    --resource-id service/spotify-cluster/spotify-track-db \
    --scalable-dimension ecs:service:DesiredCount \
    --min-capacity 1 \
    --max-capacity 5
```

#### 4. **CloudWatch Alarms**
Get notified when things go wrong

```bash
# Alert when no tasks are running
aws cloudwatch put-metric-alarm \
    --alarm-name spotify-track-db-down \
    --metric-name RunningTaskCount \
    --namespace ECS/Service \
    --statistic Average \
    --period 60 \
    --evaluation-periods 2 \
    --threshold 1 \
    --comparison-operator LessThanThreshold
```

#### 5. **CI/CD Pipeline**
Auto-deploy on git push

**Options**:
- **GitHub Actions**: Free for public repos
- **AWS CodePipeline**: Integrates with AWS services
- **GitLab CI**: If using GitLab

**Example GitHub Actions workflow**:
```yaml
name: Deploy to ECS
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v1
      - name: Build and deploy
        run: ./deploy-to-ecs.sh
```

#### 6. **Database Backups**
Neon likely handles this, but verify:
- Daily automated backups
- Point-in-time recovery
- Backup retention policy

#### 7. **Monitoring & Observability**
- **CloudWatch Dashboards**: Visualize metrics
- **X-Ray**: Distributed tracing
- **Third-party**: DataDog, New Relic, Sentry

#### 8. **Environment Separation**
- Separate dev/staging/production environments
- Different AWS accounts or separate clusters
- Use tags to track environments

---

## Key Takeaways

### What You Learned

1. **Docker Containerization**:
   - Multi-stage builds for smaller images
   - Cross-platform builds (arm64 → amd64)
   - Environment variable injection

2. **AWS Core Services**:
   - ECR: Docker registry
   - ECS: Container orchestration
   - Fargate: Serverless compute
   - IAM: Permissions and roles
   - Secrets Manager: Secure credential storage
   - VPC & Security Groups: Networking

3. **DevOps Best Practices**:
   - Infrastructure as Code (task definitions, scripts)
   - Secrets management (never commit credentials)
   - Logging and monitoring
   - Zero-downtime deployments

4. **Cloud Architecture Concepts**:
   - High availability across AZs
   - Stateless application design
   - 12-factor app principles

### Command Reference

Save these for quick access:

```bash
# Deploy new version
./deploy-to-ecs.sh

# View logs
aws logs tail /ecs/spotify-track-db --follow --region us-east-2

# Check service health
aws ecs describe-services --cluster spotify-cluster --services spotify-track-db --region us-east-2

# Get public IP
TASK_ARN=$(aws ecs list-tasks --cluster spotify-cluster --service-name spotify-track-db --region us-east-2 --query 'taskArns[0]' --output text) && \
ENI_ID=$(aws ecs describe-tasks --cluster spotify-cluster --tasks $TASK_ARN --region us-east-2 --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text) && \
aws ec2 describe-network-interfaces --network-interface-ids $ENI_ID --region us-east-2 --query 'NetworkInterfaces[0].Association.PublicIp' --output text

# Stop service (save money)
aws ecs update-service --cluster spotify-cluster --service spotify-track-db --desired-count 0 --region us-east-2

# Start service again
aws ecs update-service --cluster spotify-cluster --service spotify-track-db --desired-count 1 --region us-east-2
```

---

## Glossary

| Term | Definition |
|------|------------|
| **Container** | Packaged application with all dependencies |
| **Image** | Snapshot of a container (like a class in OOP) |
| **Task** | Running instance of a container (like an object in OOP) |
| **Service** | Maintains desired number of tasks |
| **Cluster** | Logical grouping of services |
| **Fargate** | Serverless compute for containers |
| **ECR** | Docker registry hosted by AWS |
| **IAM** | Identity and Access Management |
| **ARN** | Amazon Resource Name (unique identifier) |
| **VPC** | Virtual Private Cloud (your network) |
| **Subnet** | Subdivision of a VPC |
| **Security Group** | Virtual firewall |
| **ENI** | Elastic Network Interface (virtual network card) |
| **vCPU** | Virtual CPU (1 vCPU ≈ 1 core) |

---

## Further Reading

### AWS Documentation
- [ECS Developer Guide](https://docs.aws.amazon.com/ecs/)
- [Fargate Pricing](https://aws.amazon.com/fargate/pricing/)
- [Best Practices for ECS](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)

### Community Resources
- [r/aws subreddit](https://reddit.com/r/aws)
- [AWS re:Post](https://repost.aws/) - Q&A forum
- [Awesome ECS](https://github.com/nathanpeck/awesome-ecs) - Curated list

### Video Tutorials
- [AWS ECS Tutorial (freeCodeCamp)](https://www.youtube.com/watch?v=esISkPlnxL0)
- [Containers on AWS Overview](https://www.youtube.com/watch?v=AYAh6YDXuho)

---

**Questions?** Check the [AWS_DEPLOYMENT_SESSION.md](AWS_DEPLOYMENT_SESSION.md) for session-specific details or open an issue!

**Happy deploying!** 🚀
