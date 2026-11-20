# frozen_string_literal: true

# Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
# another app, the address from MEENG_BASE_URL.
#
#   MEENG_API_KEY=mg_xxx ruby examples/send_email.rb
#
# In your own project it is `require "meeng"`; here it is read from lib/.
$LOAD_PATH.unshift File.expand_path("../lib", __dir__)
require "meeng"

meeng = Meeng::Client.new

begin
  sent = meeng.emails.send(
    {
      from: "Acme <contato@acme.com.br>",
      to: "ana@example.com",
      subject: "Bem-vinda à Acme",
      html: "<p>Olá, Ana! Sua conta está pronta.</p>",
      text: "Olá, Ana! Sua conta está pronta."
    },
    # A key makes the send safe to repeat: the same key with the same body
    # answers the first result again, and nothing is sent twice.
    idempotency_key: "boas-vindas-ana-2026-11-20"
  )
rescue Meeng::Error => e
  abort "#{e.status} #{e.code}: #{e.message}"
end

puts "e-mail na fila: #{sent[:emailId]}"
