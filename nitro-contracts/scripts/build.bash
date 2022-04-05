#!/bin/bash
if [[ $PWD == */nitro-contracts ]];
    then $npm_execpath run hardhat compile;
    else $npm_execpath run hardhat:prod compile;
fi
