#!/bin/bash

# Manual Memory API test script for Kiseki server

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "Testing Kiseki Memory API at $BASE_URL"
echo "========================================"

# Create a test character first
echo -e "\n1. Creating a test character..."
CHARACTER_JSON=$(curl -s -X POST "$BASE_URL/characters" \
  -H "Content-Type: application/json" \
  -d '{"name":"Memory Test Character"}')
echo "$CHARACTER_JSON" | jq .
CHARACTER_ID=$(echo "$CHARACTER_JSON" | jq -r .id)
echo "Created character with ID: $CHARACTER_ID"

# Create test vectors (768-dimensional dense vector)
echo -e "\n2. Preparing test vectors..."
DENSE_VECTOR=$(python3 -c "import json; print(json.dumps([0.1] * 768))")
SPARSE_VECTOR='{"0": 0.5, "10": 0.3, "100": 0.2}'

# Store first memory fragment
echo -e "\n3. Storing first memory fragment..."
STORE_DATA_1=$(cat <<EOF
{
  "content": "This is the first test memory fragment about AI and machine learning",
  "denseVector": $DENSE_VECTOR,
  "sparseVector": $SPARSE_VECTOR,
  "metadata": {
    "source": "test",
    "type": "conversation",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  }
}
EOF
)

curl -s -X PUT "$BASE_URL/characters/$CHARACTER_ID/memory" \
  -H "Content-Type: application/json" \
  -d "{\"dType\": \"text\", \"data\": $STORE_DATA_1}" \
  -w "\nHTTP Status: %{http_code}\n"

# Store second memory fragment
echo -e "\n4. Storing second memory fragment..."
DENSE_VECTOR_2=$(python3 -c "import json; print(json.dumps([0.2] * 768))")
STORE_DATA_2=$(cat <<EOF
{
  "content": "This is the second test memory fragment about natural language processing",
  "denseVector": $DENSE_VECTOR_2,
  "sparseVector": $SPARSE_VECTOR,
  "metadata": {
    "source": "test",
    "type": "conversation",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  }
}
EOF
)

curl -s -X PUT "$BASE_URL/characters/$CHARACTER_ID/memory" \
  -H "Content-Type: application/json" \
  -d "{\"dType\": \"text\", \"data\": $STORE_DATA_2}" \
  -w "\nHTTP Status: %{http_code}\n"

# Store third memory fragment
echo -e "\n5. Storing third memory fragment..."
DENSE_VECTOR_3=$(python3 -c "import json; print(json.dumps([0.15] * 768))")
STORE_DATA_3=$(cat <<EOF
{
  "content": "This is the third test memory fragment about deep learning and neural networks",
  "denseVector": $DENSE_VECTOR_3,
  "sparseVector": $SPARSE_VECTOR,
  "metadata": {
    "source": "test",
    "type": "note",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  }
}
EOF
)

curl -s -X PUT "$BASE_URL/characters/$CHARACTER_ID/memory" \
  -H "Content-Type: application/json" \
  -d "{\"dType\": \"text\", \"data\": $STORE_DATA_3}" \
  -w "\nHTTP Status: %{http_code}\n"

# Wait for indexing
echo -e "\nWaiting 2 seconds for Qdrant indexing..."
sleep 2

# Search for memory fragments
echo -e "\n6. Searching for memory fragments..."
SEARCH_DATA=$(cat <<EOF
{
  "query": "machine learning and AI",
  "denseVector": $DENSE_VECTOR,
  "sparseVector": $SPARSE_VECTOR,
  "limit": 5
}
EOF
)

# URL encode the search data
SEARCH_DATA_ENCODED=$(echo "$SEARCH_DATA" | jq -c . | python3 -c "import sys, urllib.parse; print(urllib.parse.quote(sys.stdin.read()))")

echo "Searching with query: 'machine learning and AI'"
curl -s "$BASE_URL/characters/$CHARACTER_ID/memory?dType=text&data=$SEARCH_DATA_ENCODED" | jq .

# Search with different query
echo -e "\n7. Searching with different query..."
SEARCH_DATA_2=$(cat <<EOF
{
  "query": "neural networks",
  "denseVector": $DENSE_VECTOR_3,
  "sparseVector": $SPARSE_VECTOR,
  "limit": 3
}
EOF
)

SEARCH_DATA_2_ENCODED=$(echo "$SEARCH_DATA_2" | jq -c . | python3 -c "import sys, urllib.parse; print(urllib.parse.quote(sys.stdin.read()))")

echo "Searching with query: 'neural networks'"
curl -s "$BASE_URL/characters/$CHARACTER_ID/memory?dType=text&data=$SEARCH_DATA_2_ENCODED" | jq .

# Clean up - delete the test character
echo -e "\n8. Cleaning up - deleting test character..."
curl -s -X DELETE "$BASE_URL/characters/$CHARACTER_ID" -w "\nHTTP Status: %{http_code}\n"

echo -e "\n========================================"
echo "All Memory API tests completed!"
echo ""
echo "Summary:"
echo "- Created character: $CHARACTER_ID"
echo "- Stored 3 memory fragments"
echo "- Performed 2 search queries"
echo "- Cleaned up test data"
