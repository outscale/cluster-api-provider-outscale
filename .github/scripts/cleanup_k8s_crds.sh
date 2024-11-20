#!/bin/bash

set -e

# Function to patch resources to remove finalizers
remove_finalizers() {
    local resource_type=$1
    local resource_name=$2

    echo "Removing finalizers from $resource_type/$resource_name (if any)..."
    kubectl patch "$resource_type" "$resource_name" --type='merge' -p '{"metadata":{"finalizers":[]}}' || echo "No finalizers to patch or resource does not exist."
}

# Function to delete all resources of a given CRD type
delete_resources() {
    local resource_type=$1

    echo "Deleting all resources of type $resource_type..."
    kubectl delete "$resource_type" --all --ignore-not-found || echo "No resources of type $resource_type found."
}

# Function to delete a CRD
delete_crd() {
    local crd_name=$1

    echo "Deleting CRD $crd_name..."
    kubectl delete crd "$crd_name" --ignore-not-found || echo "CRD $crd_name not found."
}

# Main cleanup logic
cleanup_crd() {
    local crd_name=$1
    local resource_name=$2

    echo "Starting cleanup for CRD $crd_name and resource $resource_name..."

    # Remove finalizers from the resource (if exists)
    if [ -n "$resource_name" ]; then
        remove_finalizers "$crd_name" "$resource_name"
    fi

    # Delete all resources associated with the CRD
    delete_resources "$crd_name"

    # Delete the CRD itself
    delete_crd "$crd_name"

    echo "Cleanup for $crd_name complete."
}

# List of CRDs to clean up (add more as needed)
CRD_LIST=(
    "oscclusters.infrastructure.cluster.x-k8s.io"
    # Add more CRDs here if needed
)

# List of specific resources to patch/remove finalizers (CRD/resource name pairs)
RESOURCE_LIST=(
    "oscclusters.infrastructure.cluster.x-k8s.io/cluster-api-test"
    # Add more resources here if needed in the format "crd/resource_name"
)

# Perform cleanup for each resource in the RESOURCE_LIST
for resource_entry in "${RESOURCE_LIST[@]}"; do
    IFS="/" read -r crd resource <<< "$resource_entry"
    cleanup_crd "$crd" "$resource"
done

# Perform cleanup for all CRDs in the CRD_LIST (general cleanup)
for crd in "${CRD_LIST[@]}"; do
    cleanup_crd "$crd" ""
done

echo "Kubernetes CRD cleanup complete."