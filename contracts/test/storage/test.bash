#!/bin/bash
for CONTRACTNAME in Bridge Inbox Outbox RollupCore RollupUserLogic RollupAdminLogic SequencerInbox ChallengeManager
do
    echo "Checking storage change of $CONTRACTNAME"
    [ -f "./test/storage/$CONTRACTNAME.svg" ] && mv "./test/storage/$CONTRACTNAME.svg" "./test/storage/$CONTRACTNAME-old.svg"
    yarn sol2uml storage ./ -c "$CONTRACTNAME" -o "./test/storage/$CONTRACTNAME.svg"
    diff "./test/storage/$CONTRACTNAME-old.svg" "./test/storage/$CONTRACTNAME.svg"
    if [[ $? != "0" ]]
    then
        CHANGED=1
    fi
done
if [[ $CHANGED == 1 ]]
then
    exit 1
fi