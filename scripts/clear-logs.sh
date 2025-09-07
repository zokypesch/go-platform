#!/bin/bash

# Script to clear Loki and Tempo data

echo "🗑️  Clearing observability data..."

# Stop services
echo "⏹️  Stopping Loki and Tempo..."
docker-compose stop loki tempo

# Clear Loki data
echo "🧹 Clearing Loki logs data..."
rm -rf ./data/loki/*
mkdir -p ./data/loki/chunks ./data/loki/rules

# Clear Tempo data  
echo "🧹 Clearing Tempo traces data..."
rm -rf ./data/tempo/*
mkdir -p ./data/tempo/blocks

# Start services
echo "▶️  Starting Loki and Tempo..."
docker-compose start loki tempo

echo "✅ Data cleared successfully!"
echo ""
echo "📊 Services status:"
docker-compose ps loki tempo