# frozen_string_literal: true

# Checks a webhook the way an endpoint does: the signature and the time are
# verified against the raw body, and then the event is read.
# The request here is made in the program; a real endpoint reads the body and the
# headers of the request it receives, and the secret from its settings.
#
#   ruby examples/verify_webhook.rb
$LOAD_PATH.unshift File.expand_path("../lib", __dir__)
require "json"
require "meeng"
require "openssl"

secret = "whsec_exemplo"
body = JSON.generate(
  id: "call_1", type: "email.delivered", version: "2026-01-18", createdAt: "2026-11-20T12:00:00Z", teamId: 7, attempt: 1,
  data: { id: "em_1", status: "DELIVERED", from: "contato@acme.com.br", to: ["ana@example.com"], occurredAt: "2026-11-20T12:00:00Z" }
)
timestamp = (Time.now.to_f * 1000).to_i.to_s
headers = {
  "x-meeng-signature" => "v1=#{OpenSSL::HMAC.hexdigest('SHA256', secret, "#{timestamp}.#{body}")}",
  "x-meeng-timestamp" => timestamp
}

begin
  event = Meeng::Webhooks.new(secret).construct_event(body, headers: headers)
rescue Meeng::WebhookVerificationError => e
  abort "#{e.code}: #{e.message}"
end

if event[:type] == "email.delivered"
  puts "e-mail #{event[:data][:id]} entregue a #{event[:data][:to].join(', ')}"
else
  puts "evento #{event[:type]}"
end
