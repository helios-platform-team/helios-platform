#!/usr/bin/env bash

trim_ws() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

read_env_value() {
  local env_file="$1"
  local wanted_key="$2"
  local line key value

  while IFS= read -r line || [[ -n "$line" ]]; do
    line="$(trim_ws "$line")"
    [[ -z "$line" || "${line:0:1}" == "#" ]] && continue

    if [[ "$line" == export\ * ]]; then
      line="${line#export }"
      line="$(trim_ws "$line")"
    fi

    [[ "$line" != *=* ]] && continue

    key="$(trim_ws "${line%%=*}")"
    value="$(trim_ws "${line#*=}")"

    if [[ "$key" != "$wanted_key" ]]; then
      continue
    fi

    if [[ "$value" == \"*\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' ]]; then
      value="${value:1:${#value}-2}"
    fi

    printf '%s' "$value"
    return 0
  done < "$env_file"

  return 1
}