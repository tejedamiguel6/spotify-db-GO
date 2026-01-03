#!/bin/bash

# Spotify Track DB - AWS Secrets Setup Script
# This script creates the required secrets in AWS Secrets Manager

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔐 AWS Secrets Manager Setup${NC}"
echo -e "${BLUE}============================${NC}\n"

REGION="us-east-2"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}❌ .env file not found!${NC}"
    echo -e "${YELLOW}Please create a .env file with your secrets first.${NC}"
    exit 1
fi

# Load environment variables from .env using a more reliable method
while IFS='=' read -r key value; do
    # Skip comments and empty lines
    [[ $key =~ ^#.*$ ]] && continue
    [[ -z $key ]] && continue

    # Remove leading/trailing whitespace and quotes
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | xargs | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//")

    # Export the variable
    export "$key=$value"
done < .env

# Validate that required variables are loaded
if [ -z "$DATABASE_URL" ] || [ -z "$SPOTIFY_CLIENT_ID" ] || [ -z "$SPOTIFY_CLIENT_SECRET" ]; then
    echo -e "${RED}❌ Error: Required environment variables not found in .env file${NC}"
    echo -e "${YELLOW}Please ensure your .env file contains:${NC}"
    echo -e "  - DATABASE_URL"
    echo -e "  - SPOTIFY_CLIENT_ID"
    echo -e "  - SPOTIFY_CLIENT_SECRET"
    echo -e "\n${YELLOW}Current values:${NC}"
    echo -e "  DATABASE_URL: ${DATABASE_URL:-NOT SET}"
    echo -e "  SPOTIFY_CLIENT_ID: ${SPOTIFY_CLIENT_ID:-NOT SET}"
    echo -e "  SPOTIFY_CLIENT_SECRET: ${SPOTIFY_CLIENT_SECRET:-NOT SET}"
    exit 1
fi

echo -e "${GREEN}✅ Loaded environment variables from .env${NC}\n"

echo -e "${YELLOW}📋 This script will create the following secrets in AWS Secrets Manager:${NC}"
echo -e "  1. spotify-track-db/DATABASE_URL"
echo -e "  2. spotify-track-db/SPOTIFY_CLIENT_ID"
echo -e "  3. spotify-track-db/SPOTIFY_CLIENT_SECRET\n"

read -p "Do you want to continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Aborted.${NC}"
    exit 0
fi

# Function to create or update secret
create_or_update_secret() {
    local secret_name=$1
    local secret_value=$2

    echo -e "${YELLOW}Processing $secret_name...${NC}"

    if aws secretsmanager describe-secret --secret-id "$secret_name" --region $REGION > /dev/null 2>&1; then
        echo -e "${YELLOW}Secret exists. Updating...${NC}"
        aws secretsmanager put-secret-value \
            --secret-id "$secret_name" \
            --secret-string "$secret_value" \
            --region $REGION > /dev/null
        echo -e "${GREEN}✅ Updated: $secret_name${NC}"
    else
        echo -e "${YELLOW}Creating new secret...${NC}"
        aws secretsmanager create-secret \
            --name "$secret_name" \
            --secret-string "$secret_value" \
            --region $REGION > /dev/null
        echo -e "${GREEN}✅ Created: $secret_name${NC}"
    fi
}

# Create/update secrets
echo -e "\n${BLUE}Creating secrets...${NC}\n"

create_or_update_secret "spotify-track-db/DATABASE_URL" "$DATABASE_URL"
create_or_update_secret "spotify-track-db/SPOTIFY_CLIENT_ID" "$SPOTIFY_CLIENT_ID"
create_or_update_secret "spotify-track-db/SPOTIFY_CLIENT_SECRET" "$SPOTIFY_CLIENT_SECRET"

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}✅ All secrets configured successfully!${NC}"
echo -e "${GREEN}========================================${NC}\n"

echo -e "${BLUE}📋 Secret ARNs:${NC}"
aws secretsmanager describe-secret --secret-id "spotify-track-db/DATABASE_URL" --region $REGION --query 'ARN' --output text
aws secretsmanager describe-secret --secret-id "spotify-track-db/SPOTIFY_CLIENT_ID" --region $REGION --query 'ARN' --output text
aws secretsmanager describe-secret --secret-id "spotify-track-db/SPOTIFY_CLIENT_SECRET" --region $REGION --query 'ARN' --output text

echo -e "\n${GREEN}🎉 Secrets setup complete!${NC}"
echo -e "${YELLOW}💡 Note: Make sure your ECS task execution role has permission to access these secrets.${NC}"
