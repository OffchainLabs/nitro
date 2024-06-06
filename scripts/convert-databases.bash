#!/usr/bin/env bash

set -e

DEFAULT_DBCONV=/usr/local/bin/dbconv
DEFAULT_SRC=/home/user/.arbitrum/arb1/nitro

dbconv=$DEFAULT_DBCONV
src=$DEFAULT_SRC
dst=
force=false
skip_existing=false

checkMissingValue () {
    if [[ $1 -eq 0 || $2 == -* ]]; then
		echo "missing $3 argument value"
		exit 1
	fi
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --dbconv)
            shift
			checkMissingValue $# "$1" "--dbconv"
			dbconv=$1
            shift
            ;;
		--src)
            shift
			checkMissingValue $# "$1" "--src"
			src=$1
            shift
			;;
		--dst)
            shift
			checkMissingValue $# "$1" "--dst"
			dst=$1
            shift
			;;
		--force)
			force=true
			shift
			;;
		--skip-existing)
			skip_existing=true
			shift
			;;
        *)
            echo Usage: $0 \[OPTIONS..\]
            echo
            echo OPTIONS:
			echo "--dbconv          dbconv binary path (default: \"$DEFAULT_DBCONV\")"
			echo "--src             root directory containinig source databases (default: \"$DEFAULT_SRC\")"
			echo "--dst             destination path"
			echo "--force           remove destination directory if it exists"
			echo "--skip-existing   skip convertion of databases which directories already exist in the destination directory"
            exit 0
    esac
done

if ! [ -e "$dbconv" ]; then
	echo Error: Invalid dbconv binary path: "$dbconv" does not exist
	exit 1
fi

if ! [ -d "$src" ]; then
	echo Error: Invalid source path: "$src" is not a directory
	exit 1
fi

if ! [ -d $src/l2chaindata/ancient ]; then
	echo Error: Invalid ancient path: $src/l2chaindata/ancient is not a directory
fi

src=$(realpath $src)
if [ -e "$dst" ] && ! $skip_existing; then
	if $force; then
		echo == Warning! Destination already exists, --force is set, this will remove all files under the path: "$dst"
		read -p "are you sure? [y/n]" -n 1 response
		echo
		if [[ $response == "y" ]] || [[ $response == "Y" ]]; then
			(set -x; rm -r "$dst" || exit 1)
		else
			exit 0
		fi
	else
		echo Error: invalid destination path: "$dst" already exists
		exit 1
	fi
fi

if ! [ -e $dst/l2chaindata ]; then
	echo "== Converting l2chaindata db"
	(set -x; $dbconv --src.db-engine="leveldb" --src.data $src/l2chaindata --dst.db-engine="pebble" --dst.data $dst/l2chaindata --convert --compact)
	echo "== Copying l2chaindata freezer"
	(set -x; cp -r $src/l2chaindata/ancient $dst/l2chaindata/)
else
	if $skip_existing; then
		echo "== l2chaindata directory already exists, skipping conversion (--skip-existing flag is set)"
	else
		# unreachable, we already had to remove root directory
		exit 1
	fi
fi

echo

if ! [ -e $dst/arbitrumdata ]; then
	echo "== Converting arbitrumdata db"
	(set -x; $dbconv --src.db-engine="leveldb" --src.data $src/arbitrumdata --dst.db-engine="pebble" --dst.data $dst/arbitrumdata --convert --compact)
else
	if $skip_existing; then
		echo "== arbitrumdata directory already exists, skipping conversion (--skip-existing flag is set)"
	else
		# unreachable, we already had to remove root directory
		exit 1
	fi
fi

echo
if [ -e $src/wasm ]; then
	if ! [ -e $dst/wasm ]; then
		echo "== Converting wasm db"
		(set -x; $dbconv --src.db-engine="leveldb" --src.data $src/wasm --dst.db-engine="pebble" --dst.data $dst/wasm --convert --compact)
	else
		if $skip_existing; then
			echo "== wasm directory already exists, skipping conversion (--skip-existing flag is set)"
		else
			# unreachable, we already had to remove root directory
			exit 1
		fi
	fi
else
	echo "== Warning! Source directory does not contain wasm database. That is expected if source database was created with nitro version older then v2.4.0-beta.1"
fi

echo "== Done."
