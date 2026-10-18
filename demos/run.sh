#!/bin/bash
# gosa/demos/run.sh — Go 特性进度 runner: main.go -> go2sa --sa -> main.sa -> sa build-exe -> diff expected.stdout
# 用法: ./run.sh [--check] [demo目录名...]   (无参数=全量)
#   --check: 只重生成 .sa 并与进仓版逐字节比对(.sa 禁止手改,只能由编译器出;供 CI/提交前自证)
# 预期拒收工程($d/expect.refused,每行一指纹):build 必须拒收且指纹全中(真 Go 本体形;无进仓 .sa,--check 同判)
set -u
set -o pipefail
export PATH=$PATH:/tmp/go/bin:/opt/zig:/usr/local/go/bin

CHECK=0
while [ "$#" -gt 0 ]; do
  case "${1:-}" in
    --check) CHECK=1; shift ;;
    --) shift; break ;;
    -*) echo "unknown flag: $1" >&2; exit 2 ;;
    *) break ;;
  esac
done

HERE=$(dirname "$(realpath "$0")")
GOSA=$(dirname "$HERE")
GO2SA_BIN=${GO2SA_BIN:-/tmp/go2sa-demo}
SA_BIN=${SA_BIN:-/content/sa_all/sci/zig-out/bin/sa}

if [ ! -x "$GO2SA_BIN" ]; then
  echo "building go2sa ..."
  (cd "$GOSA" && go build -o "$GO2SA_BIN" ./cmd/go2sa) || exit 2
elif [ -n "$(find "$GOSA/cmd/go2sa" "$GOSA/internal/go2sa" -name '*.go' -newer "$GO2SA_BIN" 2>/dev/null | head -1)" ]; then
  echo "rebuilding stale go2sa ..."
  (cd "$GOSA" && go build -o "$GO2SA_BIN" ./cmd/go2sa) || exit 2
fi

if [ "$#" -gt 0 ]; then
  DEMOS="$*"
else
  DEMOS=$(ls "$HERE")
fi
# 非 demo 条目(README.md/run.sh 等)沿旧语义静默跳过
FILTERED=""
for n in $DEMOS; do
  [ -f "$HERE/$n/main.go" ] || continue
  FILTERED="$FILTERED $n"
done
DEMOS="$FILTERED"

pass=0; fail=0; failed=""
for n in $DEMOS; do
  d="$HERE/$n"
  if [ ! -f "$d/expected.stdout" ] && [ ! -f "$d/expect.refused" ]; then echo "$n SKIP no expected.stdout"; continue; fi
  out="$d/out"
  rm -rf "$out"
  mkdir -p "$out"
  # 预期拒收工程:build 必须拒收且指纹全中(无进仓 .sa)
  if [ -f "$d/expect.refused" ]; then
    if "$GO2SA_BIN" --sa --out "$out" "$d/main.go" >"$out/go2sa.log" 2>&1; then
      echo "$n FAIL expected refusal but build passed"
      fail=$((fail+1)); failed="$failed $n"; continue
    fi
    ok=1
    while IFS= read -r fp || [ -n "$fp" ]; do
      case "$fp" in ""|\#*) continue ;; esac
      if ! grep -qF "$fp" "$out/go2sa.log" && ! grep -qF "$fp" "$out/subset-report.txt" 2>/dev/null; then
        echo "$n FAIL refusal fingerprint missing: $fp"
        ok=0; break
      fi
    done < "$d/expect.refused"
    if [ "$ok" -eq 1 ]; then echo "$n PASS refused-as-expected"; pass=$((pass+1)); else fail=$((fail+1)); failed="$failed $n"; fi
    continue
  fi
  # .sa 落 demo 根(随批进仓供检查;其余生成物进 out/(忽略)
  # --check 模式:生成到临时位,只与进仓 main.sa 比对,不碰工作区
  if [ "$CHECK" -eq 1 ]; then
    SA_OUT="$out/check.$n"
    mkdir -p "$SA_OUT"
  else
    SA_OUT="$d"
  fi
  if ! "$GO2SA_BIN" --sa --out "$SA_OUT" "$d/main.go" >"$out/go2sa.log" 2>&1; then
    echo "$n FAIL go2sa refused: $(cat "$SA_OUT"/subset-report.txt 2>/dev/null | head -1)"
    fail=$((fail+1)); failed="$failed $n"; continue
  fi
  if [ "$CHECK" -eq 1 ]; then
    if ! diff -q "$d/main.sa" "$out/check.$n/main.sa" >"$out/diff.txt" 2>&1; then
      echo "$n FAIL main.sa drift (hand edit or compiler change)"
      fail=$((fail+1)); failed="$failed $n"; continue
    fi
    echo "$n SA-CLEAN"; pass=$((pass+1)); continue
  fi
  SAI="$d/main.sa"
  if [ ! -x "$SA_BIN" ]; then
    echo "$n FAIL sa backend not found at $SA_BIN" >&2
    fail=$((fail+1)); failed="$failed $n"; continue
  fi
  BUILD_OK=0
  "$SA_BIN" build-exe "$SAI" -o "$out/demo" >"$out/build.log" 2>&1 || BUILD_OK=$?
  if [ "$BUILD_OK" -ne 0 ]; then
    echo "$n FAIL sa build-exe failed: $(head -c 200 "$out/build.log")"
    fail=$((fail+1)); failed="$failed $n"; continue
  fi
  if ! "$out/demo" >"$out/actual.stdout" 2>"$out/run.stderr"; then
    echo "$n FAIL run exit=$?: $(head -c 120 "$out/run.stderr")"
    fail=$((fail+1)); failed="$failed $n"; continue
  fi
  if ! diff -u "$d/expected.stdout" "$out/actual.stdout" >"$out/diff.txt" 2>&1; then
    echo "$n FAIL stdout mismatch"
    fail=$((fail+1)); failed="$failed $n"; continue
  fi
  echo "$n PASS"; pass=$((pass+1))
done
echo "== pass=$pass fail=$fail =="
[ -n "$failed" ] && echo "failed:$failed"
[ "$fail" -eq 0 ]
