#!/bin/bash
if [[ $PWD == */contracts ]];
    then $npm_execpath run hardhat compile;
    else $npm_execpath run hardhat:prod compile;
fi
