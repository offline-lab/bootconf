#!/bin/bash
set -euo pipefail

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/bootconf"
DATA_DIR="/data/config/bootconf"
BUILD_BIN="build/bin"

echo "=== Bootconf Installation Script ==="
echo "Boot-time configuration tool for offline systems"
echo

if [[ "${EUID}" -ne 0 ]]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

if [[ ! -f "${BUILD_BIN}/bootconf" ]]; then
    echo "Error: Binary not found in ${BUILD_BIN}/bootconf. Run 'make' first."
    exit 1
fi

echo "Installing binary..."
install -m 755 "${BUILD_BIN}/bootconf" "${INSTALL_DIR}/"
echo "Installed: ${INSTALL_DIR}/bootconf"

echo "Creating configuration directory..."
mkdir -p "${CONFIG_DIR}"

if [[ ! -f "${CONFIG_DIR}/bootconf.yaml.example" ]]; then
    if [[ -f "bootconf.yaml" ]]; then
        install -m 644 bootconf.yaml "${CONFIG_DIR}/bootconf.yaml.example"
        echo "Installed: ${CONFIG_DIR}/bootconf.yaml.example"
    fi
else
    echo "Example config already exists: ${CONFIG_DIR}/bootconf.yaml.example"
fi

if [[ ! -f "${CONFIG_DIR}/bootconf.yaml" ]]; then
    echo "Installing configuration file..."
    if [[ -f "${CONFIG_DIR}/bootconf.yaml.example" ]]; then
        install -m 640 "${CONFIG_DIR}/bootconf.yaml.example" "${CONFIG_DIR}/bootconf.yaml"
    elif [[ -f "bootconf.yaml" ]]; then
        install -m 640 bootconf.yaml "${CONFIG_DIR}/bootconf.yaml"
    fi
    echo "Installed: ${CONFIG_DIR}/bootconf.yaml"
else
    echo "Configuration file already exists: ${CONFIG_DIR}/bootconf.yaml"
fi

echo "Creating data directory..."
if [[ ! -d "${DATA_DIR}" ]]; then
    mkdir -p "${DATA_DIR}"
    chmod 755 "${DATA_DIR}"
    echo "Created: ${DATA_DIR}"
else
    echo "Data directory already exists: ${DATA_DIR}"
fi

echo
echo "=== Installation Complete ==="
echo
echo "Next steps:"
echo "  1. Edit configuration: ${CONFIG_DIR}/bootconf.yaml"
echo "  2. Run manually: bootconf run"
echo "  3. Add to boot sequence as needed"
echo
echo "To uninstall, run: sudo bash uninstall.sh"
