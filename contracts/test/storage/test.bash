#!/bin/bash
for CONTRACTNAME in Bridge Inbox Outbox RollupCore RollupUserLogic RollupAdminLogic SequencerInbox ChallengeManager
do
    echo "Checking storage change of $CONTRACTNAME"
    yarn sol2uml storage ./ -c "$CONTRACTNAME" -o "./test/storage/$CONTRACTNAME-new.svg"
    diff "./test/storage/$CONTRACTNAME-current.svg" "./test/storage/$CONTRACTNAME-new.svg"
    if [[ $? != "0" ]]
    then
        exit 1
    fi
done