# frozen_string_literal: true

# Sends up to 100 e-mails in one request.
#
#   MEENG_API_KEY=mg_xxx ruby examples/send_batch.rb
$LOAD_PATH.unshift File.expand_path("../lib", __dir__)
require "meeng"

meeng = Meeng::Client.new
people = { "Ana" => "ana@example.com", "Bruno" => "bruno@example.com", "Carla" => "carla@example.com" }

batch = people.map do |name, email|
  {
    from: "Acme <contato@acme.com.br>",
    to: email,
    subject: "Seu extrato de novembro",
    html: "<p>Olá, #{name}! Seu extrato está disponível.</p>"
  }
end

begin
  sent = meeng.emails.batch(batch, idempotency_key: "extrato-2026-11")
rescue Meeng::Error => e
  abort "#{e.status} #{e.code}: #{e.message}"
end

sent[:data].each_with_index { |email, index| puts "#{index + 1}: #{email[:emailId]}" }
