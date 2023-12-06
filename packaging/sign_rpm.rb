#!/usr/bin/env ruby

require 'open3'
require 'tmpdir'

Result = Struct.new(:stdout, :stderr, :status)

def shellout(cmd, stdin: nil)
  stdout, stderr, status = Open3.capture3(cmd, stdin_data: stdin)
  Result.new(stdout, stderr, status)
end

def shellout!(cmd, stdin: nil)
  result = shellout(cmd, stdin: stdin)
  raise "cmd failed: #{result.status}" unless result.status.success?

  result.stdout
end

def require_env(name)
  val = ENV[name]
  raise "#{name} unset but required" if val.nil? || val.empty?

  val
end

gpg = if shellout('which gpg2').status.success?
        'gpg2'
      elsif shellout('which gpg').status.success?
        'gpg'
      else
        raise 'no gpg found'
      end

puts("GPG executable: #{gpg}")

rpm_file = File.expand_path(require_env('RPM_FILE'))
rpm_gpg_key_name = require_env('RPM_GPG_KEY_NAME')
rpm_gpg_key_ssm_name = require_env('RPM_GPG_KEY_SSM_NAME')
rpm_gpg_signing_passphrase_ssm_name = require_env('RPM_SIGNING_PASSPHRASE_SSM_NAME')

gpg_key = shellout!("aws ssm get-parameter --region us-east-1 --name #{rpm_gpg_key_ssm_name} --with-decryption --query \"Parameter.Value\" --out text")
shellout!('gpg --import --batch', stdin: gpg_key)

signing_passphrase = shellout!("aws ssm get-parameter --region us-east-1 --name #{rpm_gpg_signing_passphrase_ssm_name} --with-decryption --query \"Parameter.Value\" --out text")

Dir.mktmpdir do |tmp|
  Dir.chdir(tmp) do
    gpg_passphrase_file = File.join(tmp, 'passphrase')
    File.open(gpg_passphrase_file, 'w', 0600) do |file|
      file.write(signing_passphrase)
    end
    gpg_extra_args = ''

    rpm_gpg = shellout!("rpm --eval '%__gpg'").strip

    gpg_path = "#{ENV['HOME']}/.gnupg"
    rpmmacros = <<~MACROS
      %_signature gpg
      %_gpg_name #{rpm_gpg_key_name}
      %_gpg_path #{gpg_path}

      # Necessary since RPM 4.11 (CentOS 7), otherwise the GPG signing
      # machinery in RPM will ask for password via pinentry.
      %__gpg_sign_cmd %{__gpg} \
          gpg --pinentry-mode loopback --yes --no-tty --verbose --no-armor --batch \
          --passphrase-file #{gpg_passphrase_file} --digest-algo sha256 \
          --no-secmem-warning -u "%{_gpg_name}" -sbo %{__signature_filename} \
          %{__plaintext_filename}

      # These are SHA256 - we use them to build packages installable in FIPS mode
      %_source_filedigest_algorithm 8
      %_binary_filedigest_algorithm 8
    MACROS

    macros_path = "#{ENV['HOME']}/.rpmmacros"
    puts "Writing to #{macros_path} contents:\n#{rpmmacros}"
    File.open(macros_path, 'w') do |file|
      file.write(rpmmacros)
    end

    result = shellout("rpm --addsign #{rpm_file}")
    unless result.status.success?
      puts "command failed"
      puts "STDOUT:"
      puts result.stdout
      puts "STDERR:"
      puts result.stderr
      puts "rpmmacros (#{macros_path}):"
      puts File.read(macros_path)
      raise "cmd failed #{result.status}"
    end
  ensure
    File.delete(gpg_passphrase_file)
  end
end
