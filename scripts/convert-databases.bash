#!/usr/bin/env bash

DEFAULT_DBCONV=/usr/local/bin/dbconv
DEFAULT_SRC=/home/user/.arbitrum/arb1/nitro

dbconv=$DEFAULT_DBCONV
src=$DEFAULT_SRC
dst=
force=false
skip_existing=false

l2chaindata_status="n/a"
l2chaindata_ancient_status="n/a"
arbitrumdata_status="n/a"
wasm_status="n/a"
classicmsg_status="n/a"

checkMissingValue () {
    if [[ $1 -eq 0 || $2 == -* ]]; then
        echo "missing $3 argument value"
        exit 1
    fi
}

printStatus() {
    echo "== Conversion status:"
    echo "   l2chaindata database: $l2chaindata_status"
    echo "   l2chaindata database freezer (ancient): $l2chaindata_ancient_status"
    echo "   arbitrumdata database: $arbitrumdata_status"
    echo "   wasm database: $wasm_status"
    echo "   classic-msg database: $classicmsg_status"
}

printUsage() {
echo Usage: $0 \[OPTIONS..\]
    echo
    echo OPTIONS:
    echo "--dbconv          dbconv binary path (default: \"$DEFAULT_DBCONV\")"
    echo "--src             directory containing source databases (default: \"$DEFAULT_SRC\")"
    echo "--dst             destination directory"
    echo "--force           remove destination directory if it exists"
    echo "--skip-existing   skip convertion of databases which directories already exist in the destination directory"
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
        --help)
            printUsage
            exit 0
            ;;
        *)
            printUsage
            exit 0
    esac
done

if $force && $skip_existing; then
    echo Error: Cannot use both --force and --skipexisting
    printUsage
    exit 1
fi

if ! [ -e "$dbconv" ]; then
    echo Error: Invalid dbconv binary path: "$dbconv" does not exist
    exit 1
fi

if ! [ -n "$dst" ]; then
    echo Error: Missing destination directory \(\-\-dst\)
    printUsage
    exit 1
fi

if ! [ -d "$src" ]; then
    echo Error: Invalid source directory: \""$src"\" is missing
    exit 1
fi

src=$(realpath $src)

if ! [ -d "$src"/l2chaindata ]; then
    echo Error: Invalid source directory: \""$src"/l2chaindata\" is missing
    exit 1
fi

if ! [ -d $src/l2chaindata/ancient ]; then
    echo Error: Invalid source directory: \""$src"/l2chaindata/ancient\" is missing
    exit 1
fi

if ! [ -d "$src"/arbitrumdata ]; then
    echo Error: Invalid source directory: missing "$src/arbitrumdata" directory
    exit 1
fi

if [ -e "$dst" ] && ! $skip_existing; then
    if $force; then
        echo == Warning! Destination already exists, --force is set, this will remove all files under path: "$dst"
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

convert_result=
convert () {
    srcdir=$(echo $src/$1 | tr -s /)
    dstdir=$(echo $dst/$1 | tr -s /)
    if ! [ -e $dstdir ]; then
        echo "== Converting $1 db"
        cmd="$dbconv --src.db-engine=leveldb --src.data $srcdir --dst.db-engine=pebble --dst.data $dstdir --convert --compact"
        echo $cmd
        $cmd
        if [ $? -ne 0 ]; then
            convert_result="FAILED"
            return 1
        fi
        convert_result="converted"
        return 0
    else
        if $skip_existing; then
            echo "== Note: $dstdir directory already exists, skipping conversion (--skip-existing flag is set)"
            convert_result="skipped"
            return 0
        else
            convert_result="FAILED ($dstdir already exists)"
            return 1
        fi
    fi
}

convert "l2chaindata"
res=$?
l2chaindata_status=$convert_result
if [ $res -ne 0 ]; then
    printStatus
    exit 1
fi

if ! [ -e $dst/l2chaindata/ancient ]; then
    ancient_src=$(echo $src/l2chaindata/ancient | tr -s /)
    ancient_dst=$(echo $dst/l2chaindata/ | tr -s /)
    echo "== Copying l2chaindata ancients"
    cmd="cp -r $ancient_src $ancient_dst"
    echo $cmd
    $cmd
    if [ $? -ne 0 ]; then
        l2chaindata_ancient_status="FAILED (failed to copy)"
        printStatus
        exit 1
    fi
    l2chaindata_ancient_status="copied"
else
    if $skip_existing; then
        echo "== Note: l2chaindata/ancient directory already exists, skipping copy (--skip-existing flag is set)"
        l2chaindata_ancient_status="skipped"
    else
        # unreachable, we already had to remove root directory
        echo script error, reached unreachable
        exit 1
    fi
fi

convert "arbitrumdata"
res=$?
arbitrumdata_status=$convert_result
if [ $res -ne 0 ]; then
    printStatus
    exit 1
fi

if [ -e $src/wasm ]; then
    convert "wasm"
    res=$?
    wasm_status=$convert_result
    if [ $res -ne 0 ]; then
        printStatus
        exit 1
    fi
else
    echo "== Note: Source directory does not contain wasm database."
    wasm_status="not found in source directory"
fi

if [ -e $src/classic-msg ]; then
    convert "classic-msg"
    res=$?
    classicmsg_status=$convert_result
    if [ $res -ne 0 ]; then
        printStatus
        exit 1
    fi
else
    echo "== Note: Source directory does not contain classic-msg database."
    classicmsg_status="not found in source directory"
fi

printStatus
