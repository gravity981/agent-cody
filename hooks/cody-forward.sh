#!/bin/sh

payload=$(cat)
for url in \
	"http://127.0.0.1:8765/hooks/copilot" \
	"http://host.docker.internal:8765/hooks/copilot" \
	"http://172.17.0.1:8765/hooks/copilot"
do
	if response=$(printf '%s' "$payload" | curl -fsS --connect-timeout 0.1 -m 0.1 -H 'Expect:' --data-binary @- "$url" 2>/dev/null) &&
		[ "$response" = ok ]
	then
		break
	fi
done
exit 0
