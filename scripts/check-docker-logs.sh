#!/bin/bash

# Script to check Docker log file sizes and rotation

echo "🐳 Docker Container Logs Status"
echo "==============================="
echo ""

# Get all running containers from our compose
CONTAINERS=$(docker-compose ps --services)

for service in $CONTAINERS; do
    container_name="go-platform-${service}-1"
    
    if docker ps --format "table {{.Names}}" | grep -q "$container_name"; then
        echo "📦 $service ($container_name):"
        
        # Get container ID
        container_id=$(docker ps -f "name=$container_name" --format "{{.ID}}")
        
        if [ -n "$container_id" ]; then
            # Check log files in Docker's log directory
            log_path="/var/lib/docker/containers/$container_id"
            
            # Check if we can access the log files (may need sudo)
            if [ -d "$log_path" ] 2>/dev/null; then
                log_files=$(ls -la "$log_path"/*-json.log* 2>/dev/null | wc -l)
                if [ "$log_files" -gt 0 ]; then
                    ls -lh "$log_path"/*-json.log* 2>/dev/null | awk '{print "   " $5 " " $9}' | sed 's|.*\/||g'
                else
                    echo "   No log files found"
                fi
            else
                # Alternative: use docker logs command to get info
                echo "   Log files managed by Docker (max-size: 10m, max-file: 3)"
                log_size=$(docker logs "$container_name" 2>&1 | wc -c)
                if [ "$log_size" -gt 0 ]; then
                    echo "   Current log size: $(echo $log_size | awk '{print int($1/1024/1024)}')MB"
                else
                    echo "   Current log size: 0MB"
                fi
            fi
        else
            echo "   Container not running"
        fi
        echo ""
    fi
done

echo "📋 Log Configuration Applied:"
echo "   Max file size: 10MB"
echo "   Max files: 3"
echo "   Total max per container: 30MB"
echo ""
echo "💡 Total theoretical max: $(docker-compose ps --services | wc -l) containers × 30MB = $(($(docker-compose ps --services | wc -l) * 30))MB"