#!/bin/bash

nohup killall -9 ipfs >/dev/null 2>&1 &
nohup killall -9 ipfs-cluster-se >/dev/null 2>&1 &
echo 'Clean and ready to start'
