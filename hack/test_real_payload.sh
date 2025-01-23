curl -X POST \
  http://localhost:6969/api/webhook/harbor \
  -H 'accept-encoding: gzip' \
  -H 'content-type: application/json' \
  -H 'authorization: 029830ujsnkhs9jeoajsdiasodojius98qwejowqueo9wej' \
  -H 'content-length: 376' \
  -H 'user-agent: Go-http-client/1.1' \
  -H "X-Harbor-Event-Id: 12345-67890" \
  -d '{
  "type": "PUSH_ARTIFACT",
  "occur_at": 1737446601,
  "operator": "admin",
  "event_data": {
    "resources": [
      {
        "digest": "sha256:76525f02955aec223342400a080c73024c995c9db1ba2eabf7505cf3bdf3f1f0",
        "tag": "v1",
        "resource_url": "hb.nicolas.my.id/library/rumput:v1"
      }
    ],
    "repository": {
      "date_created": 1737446600,
      "name": "rumput",
      "namespace": "library",
      "repo_full_name": "library/rumput",
      "repo_type": "public"
    }
  }
}'

