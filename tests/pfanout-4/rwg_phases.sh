#!/bin/bash
# Resolve RWG rate/duration lists. Uses RWG_RATES/RWG_DURATIONS when set; else legacy 2-phase.

resolve_rwg_phases() {
    local base=$1 rate=$2 duration=$3
    if [ -n "$RWG_RATES" ] && [ -n "$RWG_DURATIONS" ]; then
        RESOLVED_RATES="$RWG_RATES"
        RESOLVED_DURATIONS="$RWG_DURATIONS"
    else
        RESOLVED_RATES="$base,$rate"
        RESOLVED_DURATIONS="2,$duration"
    fi
}
