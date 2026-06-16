#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLE_DIR="$ROOT/examples_mox"
DATA_DIR="$EXAMPLE_DIR/mox-localserve"
MOX_DIR="${MOX_DIR:-$ROOT/../mox}"
MOX_BIN="$DATA_DIR/bin/mox"
SIEVEMGMT_BIN="$ROOT/sievemgmt"
CA_FILE="$DATA_DIR/localhost.crt"
PID_FILE="$DATA_DIR/localserve.pid"
LOG_FILE="$DATA_DIR/localserve.log"

run_go() {
  if command -v mise >/dev/null 2>&1; then
    mise exec -- "$@"
  else
    "$@"
  fi
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: missing required command: $1" >&2
    exit 1
  fi
}

tcp_up() {
  local host=$1 port=$2
  (echo >"/dev/tcp/$host/$port") >/dev/null 2>&1
}

wait_for_tcp() {
  local host=$1 port=$2 name=$3
  for _ in {1..60}; do
    if tcp_up "$host" "$port"; then
      return 0
    fi
    sleep 1
  done
  echo "error: timed out waiting for $name on $host:$port" >&2
  exit 1
}

write_localserve_cert() {
  openssl req -x509 -nodes -newkey rsa:2048 -sha256 -days 825 \
    -keyout "$DATA_DIR/localhost.key" \
    -out "$DATA_DIR/localhost.crt" \
    -subj "/O=mox localserve/CN=localhost" \
    -addext "subjectAltName=DNS:localhost" \
    -addext "basicConstraints=critical,CA:TRUE" \
    -addext "keyUsage=critical,keyCertSign,digitalSignature,keyEncipherment" \
    -addext "extendedKeyUsage=serverAuth" >/dev/null 2>&1
  chmod 660 "$DATA_DIR/localhost.key" "$DATA_DIR/localhost.crt"
}

need_cmd go
need_cmd openssl
need_cmd python3

if [[ ! -d "$MOX_DIR" ]]; then
  echo "error: mox checkout not found at $MOX_DIR" >&2
  echo "Set MOX_DIR=/path/to/mox and rerun." >&2
  exit 1
fi

echo "Building sievemgmt..."
(cd "$ROOT" && run_go go build -o "$SIEVEMGMT_BIN" .)

mkdir -p "$DATA_DIR/bin"
echo "Building mox from $MOX_DIR..."
(cd "$MOX_DIR" && run_go go build -o "$MOX_BIN" .)

if [[ ! -f "$DATA_DIR/mox.conf" ]]; then
  echo "Initializing mox localserve data in $DATA_DIR..."
  "$MOX_BIN" localserve -dir "$DATA_DIR" -initonly
  write_localserve_cert
elif ! tcp_up localhost 5190; then
  echo "Refreshing localserve certificate..."
  write_localserve_cert
fi

if tcp_up localhost 5190; then
  echo "mox localserve is already listening on localhost:5190."
else
  echo "Starting mox localserve..."
  nohup "$MOX_BIN" localserve -dir "$DATA_DIR" >"$LOG_FILE" 2>&1 &
  echo "$!" >"$PID_FILE"
  wait_for_tcp localhost 5190 "ManageSieve"
fi
wait_for_tcp localhost 1143 "IMAP"
wait_for_tcp localhost 1587 "submission"
wait_for_tcp localhost 1080 "webmail"

echo "Creating demo mailboxes..."
python3 - <<'PY'
import imaplib

imap = imaplib.IMAP4("localhost", 1143)
imap.login("mox@localhost", "moxmoxmox")
for mailbox in ("Examples", "Lists"):
    imap.create(mailbox)
imap.logout()
PY

echo "Uploading Sieve scripts..."
export SIEVEMGMT_TLS_CA_FILE="$CA_FILE"
(cd "$EXAMPLE_DIR" && \
  "$SIEVEMGMT_BIN" --account mox upload scripts/delivery-filter.sieve delivery-filter --activate && \
  "$SIEVEMGMT_BIN" --account mox upload scripts/list-triage.sieve list-triage && \
  "$SIEVEMGMT_BIN" --account mox upload scripts/imapsieve-mark-unseen.sieve imapsieve-mark-unseen && \
  "$SIEVEMGMT_BIN" --account mox folders set Lists list-triage && \
  "$SIEVEMGMT_BIN" --account mox folders set Examples imapsieve-mark-unseen)

echo "Sending demo messages..."
python3 - <<'PY'
import email.message
import smtplib
import time

messages = [
    (
        "mox@localhost",
        "mox@localhost",
        "[sievemgmt-example] active sieve sample",
        {},
        "This message should be filed into the Examples mailbox.",
    ),
    (
        "mox@localhost",
        "mox@localhost",
        "newsletter sample",
        {"List-ID": "Example List <example.localhost>"},
        "This message should be filed into the Lists mailbox.",
    ),
    (
        "mox@localhost",
        "mox@localhost",
        "ordinary localserve message",
        {},
        "This ordinary message should stay in Inbox.",
    ),
]

with smtplib.SMTP("localhost", 1587, timeout=20) as smtp:
    smtp.ehlo("examples-mox.local")
    smtp.login("mox@localhost", "moxmoxmox")
    for sender, recipient, subject, headers, body in messages:
        msg = email.message.EmailMessage()
        msg["From"] = sender
        msg["To"] = recipient
        msg["Subject"] = subject
        msg["Message-ID"] = f"<{time.time_ns()}@examples-mox.local>"
        for name, value in headers.items():
            msg[name] = value
        msg.set_content(body)
        smtp.send_message(msg, from_addr=sender, to_addrs=[recipient])
PY

echo
echo "Done. Inspect the result in mox webmail:"
echo "  http://localhost:1080/webmail/"
echo "  login:    mox@localhost"
echo "  password: moxmoxmox"
echo
echo "Expected mailboxes:"
echo "  Examples - message with subject [sievemgmt-example]"
echo "  Lists    - newsletter/List-ID message"
echo "  Inbox    - ordinary localserve message"
echo
echo "IMAPSIEVE folder associations:"
echo "  Examples -> imapsieve-mark-unseen"
echo "  Lists    -> list-triage"
echo
echo "ManageSieve account config:"
echo "  $EXAMPLE_DIR/sievemgmt.yaml"
echo
echo "mox data directory:"
echo "  $DATA_DIR"
if [[ -f "$PID_FILE" ]]; then
  echo
  echo "mox localserve PID: $(cat "$PID_FILE")"
  echo "Log file: $LOG_FILE"
fi
