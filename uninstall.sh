#!/bin/bash
set -euo pipefail

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/bootconf"
DATA_DIR="/data/config/bootconf"

echo "=== Bootconf Uninstallation Script ==="
echo

if [[ "${EUID}" -ne 0 ]]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

echo "Removing binary..."
if [[ -f "${INSTALL_DIR}/bootconf" ]]; then
    rm -f "${INSTALL_DIR}/bootconf"
    echo "Removed: ${INSTALL_DIR}/bootconf"
else
    echo "Binary not found: ${INSTALL_DIR}/bootconf"
fi

if [[ -d "${CONFIG_DIR}" ]]; then
    read -p "Remove configuration directory ${CONFIG_DIR}? [y/N] " -n 1 -r
    echo
    if [[ ${REPLY} =~ ^[Yy]$ ]]; then
        rm -rf "${CONFIG_DIR}"
        echo "Removed: ${CONFIG_DIR}"
    else
        echo "Keeping: ${CONFIG_DIR}"
    fi
fi

if [[ -d "${DATA_DIR}" ]]; then
    read -p "Remove data directory ${DATA_DIR}? [y/N] " -n 1 -r
    echo
    if [[ ${REPLY} =~ ^[Yy]$ ]]; then
        rm -rf "${DATA_DIR}"
        echo "Removed: ${DATA_DIR}"
    else
        echo "Keeping: ${DATA_DIR}"
    fi
fi

echo
echo "=== Uninstallation Complete ==="
echo
echo "Note: Some files may have been kept based on your choices"
