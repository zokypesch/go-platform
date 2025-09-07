#!/bin/bash

# Script to check data usage of Loki and Tempo

echo "📊 Observability Data Usage Report"
echo "=================================="
echo ""

# Check if data directories exist
if [ -d "./data/loki" ]; then
    LOKI_SIZE=$(du -sh ./data/loki 2>/dev/null | cut -f1)
    LOKI_FILES=$(find ./data/loki -type f 2>/dev/null | wc -l)
    echo "📝 Loki Logs:"
    echo "   Size: $LOKI_SIZE"
    echo "   Files: $LOKI_FILES"
else
    echo "📝 Loki Logs: Directory not found"
fi

echo ""

if [ -d "./data/tempo" ]; then
    TEMPO_SIZE=$(du -sh ./data/tempo 2>/dev/null | cut -f1)
    TEMPO_FILES=$(find ./data/tempo -type f 2>/dev/null | wc -l)
    echo "🔍 Tempo Traces:"
    echo "   Size: $TEMPO_SIZE"  
    echo "   Files: $TEMPO_FILES"
else
    echo "🔍 Tempo Traces: Directory not found"
fi

echo ""
echo "💾 Total observability data:"
if [ -d "./data" ]; then
    TOTAL_SIZE=$(du -sh ./data 2>/dev/null | cut -f1)
    echo "   Size: $TOTAL_SIZE"
else
    echo "   Data directory not found"
fi

echo ""
echo "🕐 Data retention: 2 weeks (336 hours)"
echo "🧹 To clear data: ./scripts/clear-logs.sh"