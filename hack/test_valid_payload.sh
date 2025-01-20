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
          "tag": "v1.0.0",
          "digest": "sha256:12345abcdef67890"
        }
      ],
      "repository": {
        "name": "my-app",
        "repo_full_name": "my-project/my-app",
        "repo_type": "private"
      }
    }
  }'

