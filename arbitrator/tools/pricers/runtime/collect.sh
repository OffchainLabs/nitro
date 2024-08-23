cargo build --release
while true; do
    #sed -i '$d' $1
    ../target/release/runtime record --path $1 --limit 1m --filter 2s
    #sleep 1
done
