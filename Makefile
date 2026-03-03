create_session:
	grpcurl -plaintext \
	-d '{}' \
	localhost:50051 \
	pact.telegram.v1.TelegramService/CreateSession

submit_password:
	grpcurl -plaintext \
  -d '{
        "sessionId": "PASTE_YOUR_SESSION_ID",
        "password": "PASTE_YOUR_PASSWORD"
      }' \
  localhost:50051 \
  pact.telegram.v1.TelegramService/SubmitPassword

subscribe_messages:
	grpcurl -plaintext \
  -d '{
        "session_id": "PASTE_YOUR_SESSION_ID"
      }' \
  localhost:50051 \
  pact.telegram.v1.TelegramService/SubscribeMessages

delete_session:
	grpcurl -plaintext \
  -d '{
        "session_id": "PASTE_YOUR_SESSION_ID"
      }' \
  localhost:50051 \
  pact.telegram.v1.TelegramService/DeleteSession