#!/bin/bash

sequencer_url=""
caff_node_url=""
address=""
time_interval=10
allow_delay_times=10

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --sequencer) sequencer_url="$2"; shift ;;
        --caff-node) caff_node_url="$2"; shift ;;
        --address) address="$2"; shift ;;
        --interval) time_interval="$2"; shift ;;
        --allow-delay) allow_delay_times="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

if [ -z "$sequencer_url" ]; then
    echo "--sequencer is required"
    exit 1
fi

if [ -z "$caff_node_url" ]; then
    echo "--caff-node is required"
    exit 1
fi

if [ -z "$address" ]; then
    echo "--address is required"
    exit 1
fi

delay_count=0

while true; do
    current_time=$(date +"%Y-%m-%d %H:%M:%S")
    sequencer_balance=$(cast balance $address --rpc-url $sequencer_url)
    echo "[$current_time] sequencer_balance: $sequencer_balance"

    caff_node_balance=$(cast balance $address --rpc-url $caff_node_url)
    echo "[$current_time] caff_node_balance: $caff_node_balance"

    if [ "$sequencer_balance" != "$caff_node_balance" ]; then
        delay_count=$((delay_count + 1))
        echo "Warning: Balances do not match! Delay count: $delay_count"

        if [ "$delay_count" -gt "$allow_delay_times" ]; then
            echo "Error: Allowable delay times exceeded!"
            exit 1
        fi
    else
        delay_count=0
    fi

    sleep $time_interval
done
