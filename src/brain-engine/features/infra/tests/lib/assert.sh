#!/usr/bin/env bash

TEST_FAILURES=0

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  TEST_FAILURES=$((TEST_FAILURES + 1))
}

pass() {
  printf 'PASS: %s\n' "$*"
}

assert_success() {
  local status="$1"
  local message="$2"
  if [[ "$status" -eq 0 ]]; then
    pass "$message"
  else
    fail "$message (status=$status)"
  fi
}

assert_failure() {
  local status="$1"
  local message="$2"
  if [[ "$status" -ne 0 ]]; then
    pass "$message"
  else
    fail "$message (unexpected success)"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    pass "$message"
  else
    fail "$message"
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    fail "$message"
  else
    pass "$message"
  fi
}

finish_tests() {
  if [[ "$TEST_FAILURES" -ne 0 ]]; then
    printf '%s test assertion(s) failed\n' "$TEST_FAILURES" >&2
    return 1
  fi
}
