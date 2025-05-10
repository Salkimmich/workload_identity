#!/bin/bash

# Wait for Keycloak to be ready
echo "Waiting for Keycloak to be ready..."
until curl -s http://localhost:8081/health/ready; do
    sleep 5
done

# Get admin token
echo "Getting admin token..."
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8081/realms/master/protocol/openid-connect/token \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=admin" \
    -d "password=admin" \
    -d "grant_type=password" \
    -d "client_id=admin-cli" | jq -r '.access_token')

# Create realm
echo "Creating realm..."
curl -s -X POST http://localhost:8081/admin/realms \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "realm": "demo",
        "enabled": true
    }'

# Create client
echo "Creating client..."
curl -s -X POST http://localhost:8081/admin/realms/demo/clients \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "clientId": "demo-client",
        "secret": "demo-secret",
        "redirectUris": ["http://localhost:8080/auth/callback"],
        "publicClient": false,
        "directAccessGrantsEnabled": true,
        "serviceAccountsEnabled": true
    }'

# Create roles
echo "Creating roles..."
curl -s -X POST http://localhost:8081/admin/realms/demo/roles \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name": "user"}'

curl -s -X POST http://localhost:8081/admin/realms/demo/roles \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name": "admin"}'

# Create users
echo "Creating users..."
# Regular user
curl -s -X POST http://localhost:8081/admin/realms/demo/users \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "username": "user",
        "enabled": true,
        "credentials": [{
            "type": "password",
            "value": "password",
            "temporary": false
        }]
    }'

# Admin user
curl -s -X POST http://localhost:8081/admin/realms/demo/users \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "username": "admin",
        "enabled": true,
        "credentials": [{
            "type": "password",
            "value": "password",
            "temporary": false
        }]
    }'

# Get user IDs
USER_ID=$(curl -s http://localhost:8081/admin/realms/demo/users?username=user \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')
ADMIN_ID=$(curl -s http://localhost:8081/admin/realms/demo/users?username=admin \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

# Assign roles
echo "Assigning roles..."
# Get role IDs
USER_ROLE_ID=$(curl -s http://localhost:8081/admin/realms/demo/roles/user \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.id')
ADMIN_ROLE_ID=$(curl -s http://localhost:8081/admin/realms/demo/roles/admin \
    -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.id')

# Assign user role to regular user
curl -s -X POST http://localhost:8081/admin/realms/demo/users/$USER_ID/role-mappings/realm \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "[{\"id\":\"$USER_ROLE_ID\",\"name\":\"user\"}]"

# Assign both roles to admin user
curl -s -X POST http://localhost:8081/admin/realms/demo/users/$ADMIN_ID/role-mappings/realm \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "[{\"id\":\"$USER_ROLE_ID\",\"name\":\"user\"},{\"id\":\"$ADMIN_ROLE_ID\",\"name\":\"admin\"}]"

echo "Setup complete!" 