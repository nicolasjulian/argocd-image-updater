curl -X POST \
  http://localhost:6969/api/webhook/harbor \
  -H "Content-Type: application/json" \
  -H "X-Harbor-Event-Id: 12345-67890" \
  -d '{
    "type": "PUSH_ARTIFACT",
    "occur_at": 1672549200,
    "operator": "admin",
    "event_data": {
      "resources": [
        {
          "tag": "v100",
          "digest": "sha256:12345abcdef67890"
        }
      ],
      "repository": {
        "name": "lscr.io/linuxserver/chromium",
        "repo_full_name": "lscr.io/linuxserver",
        "repo_type": "private"
      }
    }
  }'

