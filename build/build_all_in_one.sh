#!/bin/bash

# All-in-one Docker build script
# Builds the image entirely inside containers - no host dependencies required
#
# Usage:
#   ./build/build_all_in_one.sh -n <image_name> -t <tag> [--push]
#
# Examples:
#   # Build locally only
#   ./build/build_all_in_one.sh -n dify-sandbox -t v1.0.0
#
#   # Build and push to registry
#   ./build/build_all_in_one.sh -n my-registry/dify-sandbox -t v1.0.0 --push
#
# Options:
#   -n, --name     Image name (required)
#   -t, --tag      Image tag (required)
#   -f, --file     Dockerfile path (default: docker/production-all-in-one.dockerfile)
#   --push         Push image to registry after build
#   -h, --help     Show this help message

set -e

# Default values
DOCKERFILE="docker/production-all-in-one.dockerfile"
PUSH_IMAGE=false
IMAGE_NAME=""
IMAGE_TAG=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -t|--tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -f|--file)
            DOCKERFILE="$2"
            shift 2
            ;;
        --push)
            PUSH_IMAGE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 -n <image_name> -t <tag> [--push]"
            echo ""
            echo "Options:"
            echo "  -n, --name     Image name (required)"
            echo "  -t, --tag      Image tag (required)"
            echo "  -f, --file     Dockerfile path (default: docker/production-all-in-one.dockerfile)"
            echo "  --push         Push image to registry after build"
            echo "  -h, --help     Show this help message"
            echo ""
            echo "Examples:"
            echo "  # Build locally only"
            echo "  $0 -n dify-sandbox -t v1.0.0"
            echo ""
            echo "  # Build and push to registry"
            echo "  $0 -n my-registry/dify-sandbox -t v1.0.0 --push"
            exit 0
            ;;
        *)
            echo "Error: Unknown option $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Validate required parameters
if [ -z "$IMAGE_NAME" ]; then
    echo "Error: Image name is required (-n or --name)"
    echo "Use -h or --help for usage information"
    exit 1
fi

if [ -z "$IMAGE_TAG" ]; then
    echo "Error: Image tag is required (-t or --tag)"
    echo "Use -h or --help for usage information"
    exit 1
fi

# Check if Dockerfile exists
if [ ! -f "$DOCKERFILE" ]; then
    echo "Error: Dockerfile not found: $DOCKERFILE"
    exit 1
fi

# Full image name with tag
FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"

echo "========================================"
echo "Dify Sandbox All-in-One Build"
echo "========================================"
echo "Image Name:  $IMAGE_NAME"
echo "Image Tag:   $IMAGE_TAG"
echo "Full Name:   $FULL_IMAGE_NAME"
echo "Dockerfile:  $DOCKERFILE"
echo "Push:        $PUSH_IMAGE"
echo "========================================"
echo ""

# Build the image
echo "Step 1: Building Docker image..."
docker build -f "$DOCKERFILE" -t "$FULL_IMAGE_NAME" .

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Build successful: $FULL_IMAGE_NAME"
else
    echo ""
    echo "✗ Build failed"
    exit 1
fi

# Push the image if requested
if [ "$PUSH_IMAGE" = true ]; then
    echo ""
    echo "Step 2: Pushing image to registry..."
    docker push "$FULL_IMAGE_NAME"
    
    if [ $? -eq 0 ]; then
        echo ""
        echo "✓ Push successful: $FULL_IMAGE_NAME"
    else
        echo ""
        echo "✗ Push failed"
        exit 1
    fi
fi

echo ""
echo "========================================"
echo "Build completed successfully!"
echo "========================================"
