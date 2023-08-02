#!/bin/bash
for CONTRACTNAME in Bridge Inbox Outbox RollupCore RollupUserLogic RollupAdminLogic SequencerInbox EdgeChallengeManager
do
    echo "Checking storage change of $CONTRACTNAME"
    [ -f "./test/storage/$CONTRACTNAME.dot" ] && mv "./test/storage/$CONTRACTNAME.dot" "./test/storage/$CONTRACTNAME-old.dot"
    yarn sol2uml storage ./ -c "$CONTRACTNAME" -o "./test/storage/$CONTRACTNAME.dot" -f dot
    diff "./test/storage/$CONTRACTNAME-old.dot" "./test/storage/$CONTRACTNAME.dot"
    if [[ $? != "0" ]]
    then
        CHANGED=1
    fi
done
if [[ $CHANGED == 1 ]]
then
    exit 1
fi