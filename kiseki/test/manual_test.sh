#!/bin/bash

# Manual API test script for Kiseki server

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "Testing Kiseki API at $BASE_URL"
echo "================================"

# Test health endpoint
echo -e "\n1. Testing health endpoint..."
curl -s "$BASE_URL/health" | jq .

# Create a character
echo -e "\n2. Creating a character..."
CHARACTER_JSON=$(curl -s -X POST "$BASE_URL/characters" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Character"}')
echo "$CHARACTER_JSON" | jq .
CHARACTER_ID=$(echo "$CHARACTER_JSON" | jq -r .id)
echo "Created character with ID: $CHARACTER_ID"

# Get the character
echo -e "\n3. Getting character by ID..."
curl -s "$BASE_URL/characters/$CHARACTER_ID" | jq .

# List all characters
echo -e "\n4. Listing all characters..."
curl -s "$BASE_URL/characters" | jq .

# Update the character
echo -e "\n5. Updating character..."
curl -s -X PUT "$BASE_URL/characters/$CHARACTER_ID" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Character"}' | jq .

# Delete the character
echo -e "\n6. Deleting character..."
curl -s -X DELETE "$BASE_URL/characters/$CHARACTER_ID" -w "\nHTTP Status: %{http_code}\n"

# Verify deletion (should get 404)
echo -e "\n7. Verifying deletion (should get 404)..."
curl -s "$BASE_URL/characters/$CHARACTER_ID" -w "\nHTTP Status: %{http_code}\n" | jq .

echo -e "\n================================"
echo "All tests completed!"
