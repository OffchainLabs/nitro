#!/bin/bash

declare -Ar exceptions=(
	[contracts]=origin/develop
	[nitro-testnode]=origin/master
)

divergent=0
for mod in `git submodule --quiet foreach 'echo $name'`; do
	branch=origin/HEAD
	if [[ -v exceptions[$mod] ]]; then
		branch=${exceptions[$mod]}
	fi

	if ! git -C $mod merge-base --is-ancestor HEAD $branch; then
		echo $mod diverges from $branch
		divergent=1
	fi
done

exit $divergent

