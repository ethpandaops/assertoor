#!/bin/bash
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "${__dir}/custom-kurtosis.devnet.config.yaml" ]; then
  args_file="${__dir}/custom-kurtosis.devnet.config.yaml"
else
  args_file="${__dir}/kurtosis.devnet.config.yaml"
fi

if [ -f "${__dir}/custom-assertoor.devnet.config.yaml" ]; then
  config_file="${__dir}/custom-assertoor.devnet.config.yaml"
else
  config_file="${__dir}/assertoor.devnet.config.yaml"
fi


## Run devnet using kurtosis
ENCLAVE_NAME="${ENCLAVE_NAME:-assertoor}"
if kurtosis enclave inspect "$ENCLAVE_NAME" > /dev/null; then
  echo "Kurtosis enclave '$ENCLAVE_NAME' is already up."
else
  kurtosis run github.com/ethpandaops/ethereum-package --enclave "$ENCLAVE_NAME" --args-file "$args_file" --non-blocking-tasks --image-download always

  # Stop assertoor instance within ethereum-package if running
  kurtosis service stop "$ENCLAVE_NAME" assertoor > /dev/null || true
fi

# Get generated configs
kurtosis files inspect "$ENCLAVE_NAME" validator-ranges validator-ranges.yaml | tail -n +2 > "${__dir}/generated-validator-ranges.yaml"
kurtosis files inspect "$ENCLAVE_NAME" assertoor-config assertoor-config.yaml | tail -n +2 > "${__dir}/generated-assertoor-config.yaml"

# Inject dev settings
export DEVNET_DIR=".hack/devnet"
cat "$config_file" | envsubst > "${__dir}/generated-assertoor-config-custom.yaml"
yq eval-all '. as $item ireduce ({}; . *+ $item)' "${__dir}/generated-assertoor-config.yaml" "${__dir}/generated-assertoor-config-custom.yaml" > "${__dir}/generated-assertoor-config-final.yaml"
mv "${__dir}/generated-assertoor-config-final.yaml" "${__dir}/generated-assertoor-config.yaml"
rm "${__dir}/generated-assertoor-config-custom.yaml"

# Resolve container hostnames to IPs for running assertoor outside Docker
# Kurtosis service hostnames need to be resolved via Docker network inspection
DOCKER_NETWORK="kt-${ENCLAVE_NAME}"

# Build a map of service hostnames to container IPs
declare -A hostname_to_ip

echo "Building hostname to IP mapping from Docker network..."
# Get all container info from the kurtosis network
# Format: container_name -> IP
while IFS= read -r line; do
  if [ -n "$line" ]; then
    container_name=$(echo "$line" | cut -d'|' -f1)
    container_ip=$(echo "$line" | cut -d'|' -f2 | cut -d'/' -f1)
    # Kurtosis container names are like: service--uuid
    # Remove the trailing --uuid to get the service name (hostname)
    service_name=$(echo "$container_name" | sed 's/--[a-f0-9]*$//')
    if [ -n "$service_name" ] && [ -n "$container_ip" ]; then
      hostname_to_ip["$service_name"]="$container_ip"
    fi
  fi
done < <(docker network inspect "$DOCKER_NETWORK" --format '{{range $id, $container := .Containers}}{{$container.Name}}|{{$container.IPv4Address}}{{"\n"}}{{end}}' 2>/dev/null)

# Extract all hostnames from URLs in the config and resolve them
echo "Resolving container hostnames to IPs..."
config_content=$(cat "${__dir}/generated-assertoor-config.yaml")

# Get unique hostnames from http:// URLs (matches pattern http://hostname:port)
hostnames=$(echo "$config_content" | grep -oE 'http://[a-zA-Z0-9_-]+:[0-9]+' | sed 's|http://||' | cut -d':' -f1 | sort -u)

for hostname in $hostnames; do
  ip="${hostname_to_ip[$hostname]}"
  if [ -n "$ip" ]; then
    echo "  $hostname -> $ip"
    config_content=$(echo "$config_content" | sed "s|http://${hostname}:|http://${ip}:|g")
  else
    echo "  WARNING: Could not resolve $hostname"
  fi
done

echo "$config_content" > "${__dir}/generated-assertoor-config.yaml"

if [ -f "${__dir}/custom-ai-config.yaml" ]; then
  ai_config_file="${__dir}/custom-ai-config.yaml"
  cat "$ai_config_file" | envsubst >> "${__dir}/generated-assertoor-config.yaml"
fi

cat <<EOF
============================================================================================================
Assertoor config at ${__dir}/generated-assertoor-config.yaml
Validator ranges at ${__dir}/generated-validator-ranges.yaml
============================================================================================================
EOF
