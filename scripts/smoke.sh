#!/bin/sh
set -eu
PORT="${SMOKE_PORT:-$((30000 + $$ % 20000))}"
LOG="${TMPDIR:-/tmp}/lightgo-smoke.log"
go run ./cmd/server -port "$PORT" >"$LOG" 2>&1 &
PID=$!
cleanup(){ kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
i=0
until curl -fsS "http://127.0.0.1:$PORT/api/stats" >/dev/null 2>&1; do
  i=$((i+1)); [ "$i" -lt 40 ] || { cat "$LOG"; exit 1; }; sleep .25
done
curl -fsS "http://127.0.0.1:$PORT/" | grep -q LightGo
curl -fsS "http://127.0.0.1:$PORT/blog" | grep -q 文章列表
curl -fsS "http://127.0.0.1:$PORT/categories" | grep -q 文章分类
curl -fsS "http://127.0.0.1:$PORT/api/blog/categories" | grep -q '"postCount"'
LOGIN=$(curl -fsS -H 'Content-Type: application/json' -d '{"username":"admin","password":"secret123"}' "http://127.0.0.1:$PORT/api/auth/login")
printf '%s' "$LOGIN" | grep -q '"code":0'
TOKEN=$(printf '%s' "$LOGIN" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"title":"Smoke Post","summary":"smoke test","content":"created by smoke test","category":"Test","tags":["smoke"],"status":"published"}' "http://127.0.0.1:$PORT/api/blog/posts" | grep -q '"code":0'
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"content":"smoke test comment"}' "http://127.0.0.1:$PORT/api/blog/posts/1/comments" | grep -q '"code":0'
curl -fsS "http://127.0.0.1:$PORT/api/blog/posts/1/comments" | grep -q 'smoke test comment'
printf '%s\n' 'smoke test passed'
