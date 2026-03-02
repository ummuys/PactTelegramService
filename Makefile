


create_session:
	grpcurl -plaintext -d '{}' localhost:50051 pact.telegram.v1.TelegramService/CreateSession

submit_password:
	grpcurl -plaintext \
  -d '{"sessionId":"459508f3-53a5-4f3a-8bcc-9314021916f4","password":"89ol5689caL!"}' \
  localhost:50051 pact.telegram.v1.TelegramService/SubmitPassword

delete_session:
	grpcurl -plaintext \
	-d '{"session_id":"6eb465d4-b0cb-4279-8c24-e835279e77be"}' \
	localhost:50051 pact.telegram.v1.TelegramService/DeleteSession

send_message:
	grpcurl -plaintext \
	-d '{"session_id":"a09b1ed6-1099-4893-ae7e-efa5672bf34f","peer":"@ummuys","text":"ура ура"}' \
	localhost:50051 pact.telegram.v1.TelegramService/SendMessage

# 	grpcurl -plaintext \
# 	-d '{"session_id":"c5e751f0-dbbe-4c97-b5dc-ec69086825bc","peer":"@irrinazakharova","text":"прога работает"}' \
# 	localhost:50051 pact.telegram.v1.TelegramService/SendMessage


subcribe_message:
	grpcurl -plaintext \
	-d '{"session_id":"a4c90b43-ad66-48c2-8086-c6a0cf53904b"}' \
	localhost:50051 pact.telegram.v1.TelegramService/SubscribeMessages



# d748f049-6e07-423d-802c-9fbac937da07 -- say hello

# c5e751f0-dbbe-4c97-b5dc-ec69086825bc

