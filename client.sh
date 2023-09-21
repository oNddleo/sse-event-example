#!/bin/bash

# Number of concurrent SSE connections you want to establish.
concurrency=50000

# URL of the SSE endpoint you want to connect to.
sse_url="http://localhost:8000/events/"

# Function to establish an SSE connection with curl.
establish_sse_connection() {
  curl -N -H "Accept: text/event-stream" $sse_url
}

# Loop to create multiple concurrent SSE connections.
for ((i = 1; i <= concurrency; i++)); do
  establish_sse_connection &
done

# Wait for all background processes to complete.
wait
