#!/bin/bash
# gosa demos gate (new pipeline: go -> sla -> sa).
#   ./run.sh [--check] [demo...]
# Always: gosla emit -> diff main.sla (byte compare; .sla is compiler output, never hand-edited).
# Refused demos (expect.refused): emit must exit 1 and every fingerprint line must appear in stderr.
# When GOSLA_SA is set (path to a sa host binary): also build-exe each main.sla and diff stdout vs expected.stdout.
set -u
HERE=$(dirname "$(realpath "$0")")
ROOT=$(dirname "$HERE")
GOSLA=${GOSLA:-$ROOT/gosla-bin}
[ -x "$GOSLA" ] || { (cd "$ROOT" && go build -o gosla-bin ./cmd/gosla) || exit 2; }
DEMOS="${@:-$(ls "$HERE" | grep -E '^[0-9]+_')}"
FAIL=0
for d in $DEMOS; do
  DIR="$HERE/$d"
  [ -d "$DIR" ] || continue
  if [ -f "$DIR/expect.refused" ]; then
    ERR=$("$GOSLA" emit "$DIR/main.go" -o /tmp/gosla_gate.sla 2>&1 >/dev/null); RC=$?
    if [ $RC -ne 1 ]; then echo "$d FAIL: expected refusal rc=1, got $RC"; FAIL=1; continue; fi
    OK=1
    while IFS= read -r fp; do
      case "$fp" in ''|\#*) continue;; esac
      case "$ERR" in *"$fp"*) ;; *) echo "$d FAIL: fingerprint missing: $fp"; OK=0;; esac
    done < "$DIR/expect.refused"
    [ $OK -eq 1 ] && echo "$d REFUSED-OK" || FAIL=1
    continue
  fi
  "$GOSLA" emit "$DIR/main.go" -o /tmp/gosla_gate.sla 2>/dev/null || { echo "$d FAIL: emit refused unexpectedly"; FAIL=1; continue; }
  if ! diff -q /tmp/gosla_gate.sla "$DIR/main.sla" > /dev/null; then echo "$d FAIL: main.sla drift"; FAIL=1; continue; fi
  if [ -n "${GOSLA_SA:-}" ]; then
    rm -rf /tmp/gosla_run; mkdir /tmp/gosla_run; cp "$DIR/main.sla" /tmp/gosla_run/
    (cd /tmp/gosla_run && SA_PLUGIN_DEV=1 "$GOSLA_SA" build-exe main.sla > build.log 2>&1) || { echo "$d FAIL: build-exe"; FAIL=1; continue; }
    OUT=$(cd /tmp/gosla_run && ./main 2>/dev/null || ./output.bin 2>/dev/null)
    EXP=$(cat "$DIR/expected.stdout")
    if [ "$OUT" == "$EXP" ]; then echo "$d PASS"; else echo "$d FAIL: stdout drift"; FAIL=1; fi
  else
    echo "$d EMIT-OK"
  fi
done
exit $FAIL
