#!/bin/bash
if [[ $PWD == */solgen ]];
    then $npm_execpath run hardhat compile;
    else $npm_execpath run hardhat:prod compile;
fi
