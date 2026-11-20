# frozen_string_literal: true

require "json"

module Meeng
  # A request the API refused.
  #
  # +code+ is the API's, such as NOT_UNIQUE, FORBIDDEN or NOT_FOUND; it is
  # INTERNAL_SERVER_ERROR when the answer carried none. +status+ is the HTTP
  # status and +retry_after+, on a 429, how many seconds to wait.
  class Error < StandardError
    REASONS = {
      400 => "Bad Request", 401 => "Unauthorized", 403 => "Forbidden", 404 => "Not Found", 409 => "Conflict",
      422 => "Unprocessable Entity", 429 => "Too Many Requests", 500 => "Internal Server Error",
      502 => "Bad Gateway", 503 => "Service Unavailable"
    }.freeze

    attr_reader :status, :code, :retry_after

    def initialize(status, code, message = "", retry_after = nil)
      @status = status
      @code = code
      @retry_after = retry_after
      super(message.empty? ? "meeng: #{status} #{code}" : "meeng: #{status} #{code}: #{message}")
    end

    # What Ruby prints for an error nobody rescued: the status and code too.
    def detailed_message(highlight: false, **)
      highlight ? "\e[1m#{self} (#{self.class.name})\e[m" : "#{self} (#{self.class.name})"
    end

    # The error a response with a status of 300 or more is: the answer's
    # +{"error": {"code", "message"}}+, +{"error": "text"}+ or the error itself.
    def self.from_response(status, headers, body)
      code = "INTERNAL_SERVER_ERROR"
      message = REASONS.fetch(status, "Error")
      data = begin
        JSON.parse(body.to_s)
      rescue JSON::ParserError
        nil
      end
      if data.is_a?(Hash)
        found = data.fetch("error", data)
        found = { "message" => found } if found.is_a?(String)
        if found.is_a?(Hash)
          code = found["code"] if found["code"].is_a?(String) && !found["code"].empty?
          message = found["message"] if found["message"].is_a?(String)
        end
      end
      retry_after = headers["retry-after"].to_s
      new(status, code, message, retry_after.match?(/\A\d+\z/) ? retry_after.to_i : nil)
    end
  end

  # A request meeng did not sign, signed too long ago, or that is not an event.
  # +code+ is MISSING_SIGNATURE, MISSING_TIMESTAMP, INVALID_SIGNATURE_FORMAT,
  # INVALID_TIMESTAMP, TIMESTAMP_OUT_OF_RANGE, SIGNATURE_MISMATCH, INVALID_BODY or
  # INVALID_JSON.
  class WebhookVerificationError < StandardError
    attr_reader :code

    def initialize(code, message)
      @code = code
      super("#{code}: #{message}")
    end
  end
end
