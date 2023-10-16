# poll the nitro endpoint until we get a 0 return code or 30mins have passed, in that case exit 1
timeout_time=$(($(date +%s) + 1800))

while (( $(date +%s) <= timeout_time )); do
  curl -X POST -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":45678,"method":"eth_chainId","params":[]}' 'http://localhost:8547'
  if [ "$?" -eq "0" ]; then
    exit 0
  else
    sleep 20
  fi
done

exit 1