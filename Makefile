


create_session:
	grpcurl -plaintext -d '{}' localhost:50051 pact.telegram.v1.TelegramService/CreateSession

submit_password:
	grpcurl -plaintext \
  -d '{"sessionId":"2f2403d2-ceb8-4e35-a09c-6a8afa14f891","password":"89ol5689caL!"}' \
  localhost:50051 pact.telegram.v1.TelegramService/SubmitPassword

delete_session:
	grpcurl -plaintext \
	-d '{"session_id":"45581b61-c641-4cdc-b191-779a6c390ea5"}' \
	localhost:50051 pact.telegram.v1.TelegramService/DeleteSession

send_message_1:
	grpcurl -plaintext \
	-d '{"session_id":"73d1e62b-b23f-4f70-a9e1-1da7aa59835d","peer":"@Sayhellotomes","text":"ура ура"}' \
	localhost:50051 pact.telegram.v1.TelegramService/SendMessage

send_message_2:
	grpcurl -plaintext \
	-d '{"session_id":"77dfcf24-6e28-4fb3-84aa-86d41bec0fa1","peer":"@ummuys","text":"ура ура"}' \
	localhost:50051 pact.telegram.v1.TelegramService/SendMessage

# 	grpcurl -plaintext \
# 	-d '{"session_id":"c5e751f0-dbbe-4c97-b5dc-ec69086825bc","peer":"@irrinazakharova","text":"прога работает"}' \
# 	localhost:50051 pact.telegram.v1.TelegramService/SendMessage


subcribe_message_1:
	grpcurl -plaintext \
	-d '{"session_id":"34a212a9-88fa-43da-b3f2-9899ebae594b"}' \
	localhost:50051 pact.telegram.v1.TelegramService/SubscribeMessages


subcribe_message_2:
	grpcurl -plaintext \
	-d '{"session_id":"77dfcf24-6e28-4fb3-84aa-86d41bec0fa1"}' \
	localhost:50051 pact.telegram.v1.TelegramService/SubscribeMessages


# d748f049-6e07-423d-802c-9fbac937da07 -- say hello

# c5e751f0-dbbe-4c97-b5dc-ec69086825bc

