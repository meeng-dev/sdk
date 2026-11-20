# frozen_string_literal: true

# Adds a sending domain, prints the DNS records to publish and asks for them to
# be checked.
#
#   MEENG_API_KEY=mg_xxx ruby examples/add_domain.rb
$LOAD_PATH.unshift File.expand_path("../lib", __dir__)
require "meeng"

meeng = Meeng::Client.new

begin
  domain = meeng.domains.create({ name: "acme.com.br", region: "us-east-1" })
  puts "publique estes registros no DNS de #{domain[:name]}"
  domain[:dnsRecords].each do |record|
    puts format("  %-5s %-40s %s", record[:type], record[:name], record[:value])
  end

  meeng.domains.verify(domain[:id])
  puts "situação: #{meeng.domains.get(domain[:id])[:status]}"
rescue Meeng::Error => e
  abort "#{e.status} #{e.code}: #{e.message}"
end
