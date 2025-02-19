#!/bin/bash
cp /home/callisto/config/config.yaml /home/callisto/.callisto/config.yaml
callisto parse genesis-file --genesis-file-path /home/callisto/.callisto/genesis.json && callisto start