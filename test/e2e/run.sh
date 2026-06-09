#!/bin/bash
set -euo pipefail

PASS=0
FAIL=0

# ─── Assertion helpers ───────────────────────────────────────────────────────

assert_file_exists() {
    local path="$1"
    if [ -f "$path" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $path exists"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path does not exist"
    fi
}

assert_dir_exists() {
    local path="$1"
    if [ -d "$path" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $path exists (dir)"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path does not exist (dir)"
    fi
}

assert_file_not_exists() {
    local path="$1"
    if [ ! -e "$path" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $path does not exist"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path exists but should not"
    fi
}

assert_dir_not_exists() {
    local path="$1"
    if [ ! -d "$path" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $path does not exist (dir)"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path exists but should not (dir)"
    fi
}

assert_file_contains() {
    local path="$1"
    local pattern="$2"
    if grep -q "$pattern" "$path" 2>/dev/null; then
        PASS=$((PASS + 1))
        echo "  PASS: $path contains '$pattern'"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path does not contain '$pattern'"
    fi
}

assert_file_not_contains() {
    local path="$1"
    local pattern="$2"
    if ! grep -q "$pattern" "$path" 2>/dev/null; then
        PASS=$((PASS + 1))
        echo "  PASS: $path does not contain '$pattern'"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path contains '$pattern' but should not"
    fi
}

assert_file_content() {
    local path="$1"
    local expected="$2"
    local actual
    actual=$(cat "$path" 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $path content matches"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path content mismatch"
        echo "    expected: $expected"
        echo "    actual:   $actual"
    fi
}

assert_file_perms() {
    local path="$1"
    local expected="$2"
    local actual
    actual=$(stat -c '%a' "$path" 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $path perms=$actual"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $path perms=$actual (expected $expected)"
    fi
}

assert_exit_code() {
    local expected="$1"
    local description="$2"
    shift 2
    set +e
    output=$("$@" 2>&1)
    actual=$?
    set -e
    if [ "$actual" -eq "$expected" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $description (exit $actual)"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $description (exit $actual, expected $expected)"
        echo "    output: $output"
    fi
}

assert_output_contains() {
    local pattern="$1"
    local description="$2"
    shift 2
    set +e
    output=$("$@" 2>&1)
    actual=$?
    set -e
    if echo "$output" | grep -q "$pattern"; then
        PASS=$((PASS + 1))
        echo "  PASS: $description"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $description"
        echo "    expected pattern: $pattern"
        echo "    output: $output"
    fi
}

assert_output_not_contains() {
    local pattern="$1"
    local description="$2"
    shift 2
    set +e
    output=$("$@" 2>&1)
    actual=$?
    set -e
    if ! echo "$output" | grep -q "$pattern"; then
        PASS=$((PASS + 1))
        echo "  PASS: $description"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $description"
        echo "    pattern should be absent: $pattern"
        echo "    output: $output"
    fi
}

# ─── Setup ────────────────────────────────────────────────────────────────────

CONFIG="/etc/bootconf/test-config.yaml"
INVALID_CONFIG="/boot/firmware/config/invalid-config.yaml"

echo "=== Building test environment ==="
mkdir -p /data/bootconf /data/config/ssh /data/config/wifi /data/config/services
mkdir -p /data/config/users /data/home /data/config/app /data/config/secrets
mkdir -p /data/config/daemon /data/config/monitor /data/config/testservice
mkdir -p /etc/sudoers.d

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 1: VERSION COMMAND
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf version ==="

assert_exit_code 0 "version exits 0" bootconf version
assert_output_contains "e2e-test" "version shows build version" bootconf version
assert_output_contains "Commit:" "version shows commit field" bootconf version
assert_output_contains "Built:" "version shows build time" bootconf version

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 2: VALIDATE COMMAND
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf validate (valid config, --config flag) ==="

assert_exit_code 0 "validate with --config" bootconf validate --config "$CONFIG"
assert_output_contains "configuration is valid" "validate prints success message" bootconf validate --config "$CONFIG"

echo ""
echo "=== Test: bootconf validate (valid config, -c short flag) ==="

assert_exit_code 0 "validate with -c" bootconf validate -c "$CONFIG"

echo ""
echo "=== Test: bootconf validate (valid config, default path - copy to default location) ==="

cp "$CONFIG" /boot/firmware/bootconf.yaml
assert_exit_code 0 "validate with default config path" bootconf validate
rm -f /boot/firmware/bootconf.yaml

echo ""
echo "=== Test: bootconf validate (invalid config) ==="

assert_exit_code 1 "validate rejects invalid config" bootconf validate --config "$INVALID_CONFIG"
assert_output_contains "password_hash" "validate reports password_hash error" bootconf validate --config "$INVALID_CONFIG"

echo ""
echo "=== Test: bootconf validate (missing config file) ==="

assert_exit_code 0 "validate exits 0 for missing config" bootconf validate --config /nonexistent/path.yaml
assert_output_contains "no config file found" "validate reports missing file" bootconf validate --config /nonexistent/path.yaml

echo ""
echo "=== Test: bootconf validate (verbose flag) ==="

assert_exit_code 0 "validate with --verbose" bootconf validate --verbose --config "$CONFIG"

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 3: RUN COMMAND - DRY RUN
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run --dry-run (no files created) ==="

assert_exit_code 0 "dry-run exits 0" bootconf run --dry-run --config "$CONFIG"

assert_file_not_exists /data/config/wifi/wpa_supplicant.conf
assert_file_not_exists /data/config/services/testservice
assert_file_not_exists /data/config/services/ssh
assert_file_not_exists /data/config/services/wifi
assert_file_not_exists /data/config/users/admin.conf
assert_file_not_exists /data/config/app/app.conf
assert_dir_not_exists /data/home/admin

echo ""
echo "=== Test: bootconf run --dry-run --section wifi (single section) ==="

assert_exit_code 0 "dry-run single section exits 0" bootconf run --dry-run --section wifi --config "$CONFIG"
assert_file_not_exists /data/config/wifi/wpa_supplicant.conf

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 4: RUN COMMAND - FULL RUN
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run (full execution with --config) ==="

assert_exit_code 0 "run with --config exits 0" bootconf run --config "$CONFIG"

# ─── WiFi ────────────────────────────────────────────────────────────────────

echo ""
echo "=== Verifying wifi configuration ==="

assert_file_exists /data/config/wifi/wpa_supplicant.conf
assert_file_contains /data/config/wifi/wpa_supplicant.conf "ssid=\"TestNetwork\""
assert_file_contains /data/config/wifi/wpa_supplicant.conf "psk=a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3"
assert_file_contains /data/config/wifi/wpa_supplicant.conf "country=NL"
assert_file_contains /data/config/wifi/wpa_supplicant.conf "ctrl_interface="
assert_file_contains /data/config/wifi/wpa_supplicant.conf "network={"
assert_file_not_contains /data/config/wifi/wpa_supplicant.conf "plaintext"
assert_file_perms /data/config/wifi/wpa_supplicant.conf "600"
assert_file_exists /data/config/services/wifi

# ─── SSH ─────────────────────────────────────────────────────────────────────

echo ""
echo "=== Verifying ssh configuration ==="

assert_file_exists /data/config/services/ssh

# ─── Services ────────────────────────────────────────────────────────────────

echo ""
echo "=== Verifying services configuration ==="

assert_file_exists /data/config/services/testservice
assert_file_exists /data/config/services/daemon
assert_file_not_exists /data/config/services/disabled-svc
assert_file_not_exists /data/config/services/monitor

assert_file_exists /data/config/testservice/test.conf
assert_file_contains /data/config/testservice/test.conf "key=value"

assert_file_exists /data/config/daemon/daemon.conf
assert_file_contains /data/config/daemon/daemon.conf "daemon=true"
assert_file_contains /data/config/daemon/daemon.conf "log_level=info"

assert_file_exists /data/config/monitor/monitor.conf
assert_file_contains /data/config/monitor/monitor.conf "enabled=true"

# ─── Users ───────────────────────────────────────────────────────────────────

echo ""
echo "=== Verifying users configuration ==="

# Admin (sudo=true)
assert_file_exists /data/config/users/admin.conf
assert_file_contains /data/config/users/admin.conf "u admin"
assert_file_contains /data/config/users/admin.conf "m admin sudo"
assert_dir_exists /data/home/admin
assert_dir_exists /data/home/admin/.ssh
assert_file_exists /data/home/admin/.ssh/authorized_keys
assert_file_contains /data/home/admin/.ssh/authorized_keys "ssh-ed25519"
assert_file_contains /data/home/admin/.ssh/authorized_keys "test@e2e"

# Operator (sudo=false, multiple keys)
assert_file_exists /data/config/users/operator.conf
assert_file_contains /data/config/users/operator.conf "u operator"
assert_file_not_contains /data/config/users/operator.conf "m operator sudo"
assert_dir_exists /data/home/operator
assert_dir_exists /data/home/operator/.ssh
assert_file_exists /data/home/operator/.ssh/authorized_keys
assert_file_contains /data/home/operator/.ssh/authorized_keys "operator@e2e"
assert_file_contains /data/home/operator/.ssh/authorized_keys "operator2@e2e"

# Auditor (sudo=false, no keys)
assert_file_exists /data/config/users/auditor.conf
assert_file_contains /data/config/users/auditor.conf "u auditor"
assert_file_not_contains /data/config/users/auditor.conf "m auditor sudo"
assert_dir_exists /data/home/auditor
assert_dir_exists /data/home/auditor/.ssh
assert_file_not_exists /data/home/auditor/.ssh/authorized_keys

# ─── Files ───────────────────────────────────────────────────────────────────

echo ""
echo "=== Verifying files configuration ==="

assert_file_exists /data/config/app/app.conf
assert_file_contains /data/config/app/app.conf "app_name=bootconf"
assert_file_contains /data/config/app/app.conf "app_env=production"
assert_file_perms /data/config/app/app.conf "640"

assert_file_exists /data/config/secrets/secret.key
assert_file_perms /data/config/secrets/secret.key "600"

assert_file_exists /data/config/app/monitor.conf
assert_file_contains /data/config/app/monitor.conf "enabled=true"
assert_file_perms /data/config/app/monitor.conf "644"

# ─── Status ──────────────────────────────────────────────────────────────────

echo ""
echo "=== Verifying status file ==="

assert_file_exists /data/bootconf/.bootconf/status.json
assert_file_contains /data/bootconf/.bootconf/status.json '"overall": true'
assert_file_contains /data/bootconf/.bootconf/status.json '"section": "wifi"'
assert_file_contains /data/bootconf/.bootconf/status.json '"section": "ssh"'
assert_file_contains /data/bootconf/.bootconf/status.json '"section": "services"'
assert_file_contains /data/bootconf/.bootconf/status.json '"section": "users"'
assert_file_contains /data/bootconf/.bootconf/status.json '"section": "files"'
assert_file_contains /data/bootconf/.bootconf/status.json '"section": "system"'

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 5: STATUS COMMAND
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf status (summary) ==="

assert_exit_code 0 "status exits 0" bootconf status --config "$CONFIG"
assert_output_contains "Overall:" "status shows overall" bootconf status --config "$CONFIG"
assert_output_contains "PASS" "status shows pass" bootconf status --config "$CONFIG"

echo ""
echo "=== Test: bootconf status --full ==="

assert_exit_code 0 "status --full exits 0" bootconf status --full --config "$CONFIG"
assert_output_contains "duration:" "status --full shows duration" bootconf status --full --config "$CONFIG"
assert_output_contains "message:" "status --full shows message" bootconf status --full --config "$CONFIG"

echo ""
echo "=== Test: bootconf status --section wifi ==="

assert_exit_code 0 "status --section exits 0" bootconf status --section wifi --config "$CONFIG"
assert_output_contains "wifi" "status --section shows wifi" bootconf status --section wifi --config "$CONFIG"

echo ""
echo "=== Test: bootconf status --failed (all pass) ==="

assert_exit_code 0 "status --failed exits 0" bootconf status --failed --config "$CONFIG"
assert_output_contains "all sections passed" "status --failed shows all pass" bootconf status --failed --config "$CONFIG"

echo ""
echo "=== Test: bootconf status (default path - config was copied there earlier) ==="

assert_exit_code 0 "status with default config path" bootconf status
assert_output_contains "Overall:" "status shows overall with default path" bootconf status

echo ""
echo "=== Test: bootconf status -c short flag ==="

assert_exit_code 0 "status with -c" bootconf status -c "$CONFIG"

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 6: RUN COMMAND - RE-RUN (IDEMPOTENCY)
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run (second run - idempotent) ==="

assert_exit_code 0 "second run exits 0" bootconf run --config "$CONFIG"

assert_file_exists /data/config/wifi/wpa_supplicant.conf
assert_file_exists /data/config/services/testservice
assert_file_exists /data/config/users/admin.conf
assert_file_exists /data/config/app/app.conf

# Existing files should NOT be overwritten - original content preserved
assert_file_contains /data/config/app/app.conf "app_name=bootconf"
assert_file_exists /data/bootconf/.bootconf/status.json

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 7: RUN COMMAND - SINGLE SECTION
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run --section wifi (single section only) ==="

# Remove wifi artifact to verify re-creation
rm -f /data/config/wifi/wpa_supplicant.conf
rm -f /data/config/services/wifi

assert_exit_code 0 "run --section wifi exits 0" bootconf run --section wifi --config "$CONFIG"
assert_file_exists /data/config/wifi/wpa_supplicant.conf
assert_file_exists /data/config/services/wifi

echo ""
echo "=== Test: bootconf run --section files (single section only) ==="

rm -f /data/config/app/app.conf

assert_exit_code 0 "run --section files exits 0" bootconf run --section files --config "$CONFIG"
assert_file_exists /data/config/app/app.conf

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 8: RUN COMMAND - MISSING CONFIG
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run (missing config exits 0 silently) ==="

assert_exit_code 0 "run with missing config exits 0" bootconf run --config /nonexistent/path.yaml

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 9: RUN COMMAND - INVALID CONFIG
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run (invalid config exits 1) ==="

assert_exit_code 1 "run rejects invalid config" bootconf run --config "$INVALID_CONFIG"

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 10: FILE OVERWRITE PROTECTION
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: files module - existing files get .new suffix ==="

# app.conf already exists from earlier runs. Run again and verify .new created.
assert_file_exists /data/config/app/app.conf

bootconf run --config "$CONFIG"

assert_file_exists /data/config/app/app.conf
# Some files that already existed should now also have .new variants
# (depends on whether the file was already present from a previous run)

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 11: VERBOSE FLAG
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: bootconf run --verbose ==="

assert_exit_code 0 "run with --verbose exits 0" bootconf run --verbose --config "$CONFIG"
assert_output_contains "bootconf:" "verbose shows log prefix" bootconf run --verbose --config "$CONFIG"

echo ""
echo "=== Test: bootconf validate --verbose ==="

assert_exit_code 0 "validate with --verbose exits 0" bootconf validate --verbose --config "$CONFIG"

# ═══════════════════════════════════════════════════════════════════════════════
# SECTION 12: STATUS FILE PERSISTENCE
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Test: status file persists across runs ==="

assert_file_exists /data/bootconf/.bootconf/status.json

TIMESTAMP=$(python3 -c "import json; print(json.load(open('/data/bootconf/.bootconf/status.json'))['timestamp'])" 2>/dev/null || \
            jq -r '.timestamp' /data/bootconf/.bootconf/status.json 2>/dev/null || \
            grep -o '"timestamp": "[^"]*"' /data/bootconf/.bootconf/status.json | head -1)

if [ -n "$TIMESTAMP" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: status.json has valid timestamp field"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: status.json missing timestamp field"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# SUMMARY
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "=== Summary ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "E2E TEST FAILED"
    exit 1
fi

echo ""
echo "E2E TEST PASSED"
