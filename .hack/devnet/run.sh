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
  kurtosis run github.com/ethpandaops/ethereum-package --enclave "$ENCLAVE_NAME" --args-file "$args_file" --image-download always --non-blocking-tasks

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

cat <<EOF
============================================================================================================
Assertoor config at ${__dir}/generated-assertoor-config.yaml
Validator ranges at ${__dir}/generated-validator-ranges.yaml
============================================================================================================
EOF
