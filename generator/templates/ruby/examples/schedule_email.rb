# frozen_string_literal: true

# Schedules an e-mail, moves it to another time and cancels it.
#
#   MEENG_API_KEY=mg_xxx ruby examples/schedule_email.rb
$LOAD_PATH.unshift File.expand_path("../lib", __dir__)
require "meeng"
require "time"

meeng = Meeng::Client.new
in_hours = ->(hours) { (Time.now.utc + (hours * 3600)).iso8601 }

begin
  sent = meeng.emails.send(
    {
      from: "Acme <contato@acme.com.br>",
      to: "ana@example.com",
      subject: "Lembrete: sua reunião é amanhã",
      html: "<p>Nos vemos às 10h.</p>",
      scheduledAt: in_hours.call(24)
    }
  )
  puts "agendado: #{sent[:emailId]}"

  meeng.emails.update(sent[:emailId], { scheduledAt: in_hours.call(48) })
  puts "remarcado"

  meeng.emails.cancel(sent[:emailId])
  puts "cancelado"
rescue Meeng::Error => e
  abort "#{e.status} #{e.code}: #{e.message}"
end
